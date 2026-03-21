# Assignment 8 - Step II: DynamoDB Implementation Notes

## Partition Key Strategy

My first instinct was to use `customer_id` as the partition key since carts belong to customers. That turned out to be the wrong call. In a real shopping scenario, a small number of active customers would generate the bulk of the traffic, meaning a few partitions would absorb almost all the writes while the rest sat idle. That is exactly the hot partition problem DynamoDB is designed to avoid.

The second thought was to just reuse the auto-increment integer from MySQL. That has the same problem in a different form. Sequential IDs land on the same partition range until DynamoDB splits it, so you still get uneven distribution at the start.

I ended up going with `cart_id` as a UUID string. UUID keys are randomly distributed across the full key space, so writes spread evenly across partitions from the very first request regardless of how many active users there are. This is the standard recommendation for high-write DynamoDB tables and it directly addresses the even distribution constraint.

I decided against adding a sort key. All three access patterns (create, retrieve, update items) operate on a single cart at a time using just the cart ID, so range queries are never needed. Adding a sort key would have added complexity without any practical benefit.

![DynamoDB Table Settings](./screenshots/dynamodb_table_settings.png)
*DynamoDB console showing partition key (cart_id), no sort key, and On-demand capacity mode*

---

## Table Structure

I went with a single table design, embedding items directly inside the cart as a Map attribute keyed by string product ID:

```
shopping-carts
  cart_id        (String, PK) - UUID
  customer_id    (Number)
  items          (Map)
    "1"  -> { product_id: 1, quantity: 5 }
    "3"  -> { product_id: 3, quantity: 2 }
```

The choice to use a Map instead of a List for items was an important one. With a Map keyed by product ID, I can do an atomic `UpdateItem` using `SET items.#pid = :item` to add or update a specific product without reading the whole item first. A List would have required a read-modify-write cycle to find the right index, which is not atomic under concurrent access.

No secondary indexes were added since the spec never requires querying carts by customer ID. A GSI would have added cost and complexity with no benefit for the given access patterns.

Capacity mode was set to `PAY_PER_REQUEST` (On-demand). This avoids over-provisioning and scales automatically, which makes sense for an assignment workload with unpredictable traffic.

### DynamoDB SDK and Attribute Formatting

One thing that took some getting used to was how DynamoDB represents data. Every attribute has an explicit type wrapper in the SDK. A string is not just a string, it is `{"S": "value"}`, a number is `{"N": "42"}`, and a map is `{"M": {...}}`. The `attributevalue` package in the AWS SDK handles the marshaling automatically when you pass Go structs, but writing raw `UpdateItem` expressions still requires working with these typed wrappers directly, for example `&types.AttributeValueMemberM{Value: itemVal}` for the item map in the update expression.

For the three endpoints, the right DynamoDB operations were fairly clear once the data model was settled. `PutItem` for cart creation since it creates a whole new item. `GetItem` for retrieval since we always fetch by partition key. `UpdateItem` with a `SET` expression for adding items to a cart, because it lets you modify a single attribute inside the item without reading and rewriting the whole thing. `Query` was never needed since all access goes through the partition key directly.

---

## Key Differences from MySQL (Step I)

| Dimension | MySQL | DynamoDB |
|---|---|---|
| Cart ID type | Auto-increment integer | UUID string |
| Item storage | Separate `shopping_cart_items` table | Embedded Map in cart item |
| Cart retrieval | JOIN across 3 tables | Single GetItem, no joins |
| Item update | INSERT ... ON DUPLICATE KEY UPDATE | UpdateItem with SET expression |
| Verify cart exists on update | SELECT before INSERT (2 round trips) | ConditionExpression on UpdateItem (1 round trip) |
| Schema enforcement | Foreign keys, CHECK constraints, transactions | Application-level validation only |
| Connection management | Connection pool (max 5, idle 2) | Stateless HTTP SDK, no pooling needed |

The most meaningful difference in practice: MySQL needed a full transaction (begin, check cart exists, check product exists, upsert, commit) to safely add an item. DynamoDB collapsed this into a single `UpdateItem` call with a `ConditionExpression` that fails atomically if the cart does not exist. Fewer round trips, simpler code.

One tradeoff: because DynamoDB has no joins, product details (SKU, manufacturer, weight) cannot be fetched from a products table. Instead, the 5 seed products are kept in memory and used to enrich cart items at read time. This works fine for a fixed product catalog but would not scale to a dynamic catalog without adding a separate products table or embedding product details into each cart item.

---

## Performance Results

### DynamoDB (150 operations, 0 failures)

| Operation | Median | Average | Min | Max |
|---|---|---|---|---|
| POST /shopping-carts | 42ms | 54ms | 31ms | 368ms |
| POST /shopping-carts/{id}/items | 45ms | 52ms | 32ms | 138ms |
| GET /shopping-carts/{id} | 36ms | 42ms | 29ms | 122ms |
| Aggregated | 42ms | 49ms | 29ms | 368ms |

![Locust Statistics](./screenshots/locust_stats.png)
*Locust statistics showing 150 completed operations with 0 failures*

![Locust Charts](./screenshots/locust_charts.png)
*RPS, response times, and user count over the test duration*

### MySQL (Step I - from teammate's implementation)

> Note: fill in actual numbers from `mysql_test_results.json`

| Operation | Median | Average | Min | Max |
|---|---|---|---|---|
| POST /shopping-carts | _ms | _ms | _ms | _ms |
| POST /shopping-carts/{id}/items | _ms | _ms | _ms | _ms |
| GET /shopping-carts/{id} | _ms | _ms | _ms | _ms |

### Comparison Observations

CloudWatch confirmed DynamoDB-level latency was 2-5ms per operation at the database layer. The 42ms end-to-end median from Locust is mostly network and ECS overhead, not database processing time. No throttling occurred during the test, which confirms that PAY_PER_REQUEST was the right capacity choice.

![DynamoDB Latency](./screenshots/dynamodb_latency.png)
*SuccessfulRequestLatency for GetItem, PutItem, UpdateItem during the test*

![Consumed Capacity](./screenshots/consumed_capacity.png)
*ConsumedReadCapacityUnits and ConsumedWriteCapacityUnits showing activity spike at 18:15*

![ECS and DynamoDB Combined](./screenshots/ecs_dynamodb_combined.png)
*ECS CPU and memory utilization alongside DynamoDB latency - ECS was barely stressed*

---

## Eventual Consistency Investigation

Three scenarios were tested to observe read-after-write consistency behavior.

| Scenario | Runs | Consistent on First Read | Avg Time-to-Consistent |
|---|---|---|---|
| Create cart, immediately GET it | 20 | 20/20 (100%) | 72ms |
| Add item, immediately GET cart items | 20 | 20/20 (100%) | 81ms |
| 10 concurrent writers, then GET | 10 | 10/10 (100%) | 93ms |

![Consistency Test Results](./screenshots/consistency_test.png)
*Terminal output from consistency_test.py showing all three scenarios*

The most surprising finding here was that eventual consistency never actually showed up. Every single read after a write returned the correct data on the first attempt, across all 50 test runs. Going into this, I expected to see at least some stale reads, especially in the concurrent writes scenario.

After some research, this makes sense for a single-region DynamoDB setup. Eventual consistency in DynamoDB is most relevant in multi-region Global Tables configurations, where writes need to replicate across regions. In a single region (us-east-1), storage nodes are close together and replication is fast enough that by the time a network request completes and a client sends a follow-up read, the data is already consistent.

For the shopping cart use case, this means you can build the application without defensive retry logic for consistency in single-region deployments. If this were a Global Tables setup with users across regions, the story would be different.

The practical takeaway for handling consistency gracefully is to design around the worst case anyway. The consistency test script was built with a retry loop that re-reads every 10ms until the expected data appears or 2 seconds elapse. Even though it never needed to retry in this test, that pattern is the right one for production: attempt the read, check if the expected data is there, and retry with a short backoff if it is not. This keeps the application correct without assuming strong consistency.

---

## NoSQL vs SQL Trade-offs

Things that worked better with DynamoDB than expected:
- No connection pool management. The AWS SDK handles HTTP connections automatically, which simplifies the application startup code.
- The `ConditionExpression` pattern for verifying cart existence before an update was cleaner than MySQL's two-query approach.
- Schema changes require zero infrastructure work. Adding a new field to a cart is just a code change.

Things that were harder than expected:
- The expression syntax (`SET #items.#pid = :item` with `ExpressionAttributeNames`) has a steep learning curve compared to SQL. Writing the UpdateItem call took more time than the equivalent MySQL transaction.
- No auto-increment means the application is responsible for generating unique IDs. This also changed the API response type from integer to string, which is a breaking change compared to Step I.
- Product enrichment had to be done in the application layer since DynamoDB cannot join tables. For a real application with thousands of products, this approach would not work without rethinking the data model.

Overall, DynamoDB was a natural fit for the shopping cart use case given the simple access patterns and the high write volume expected from cart updates. MySQL would be a better choice if the product catalog needed complex queries or if referential integrity between carts and products was a hard requirement.
