# Use the official Locust image as the base
FROM locustio/locust:latest

# Copy the rest of the application/locust files
COPY . .

# Expose the default web interface and worker communication ports
EXPOSE 8089 8089

# Set the entrypoint to locust
ENTRYPOINT ["locust"]