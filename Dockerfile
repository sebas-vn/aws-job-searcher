# Use an official Go image as the base image
FROM golang:1.23

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files first for dependency installation
COPY go.mod go.sum ./

# Download and cache dependencies
RUN go mod download

# Copy the rest of the application files
COPY . .

# Build the Go application
RUN go build -o main .

# Command to run the application
CMD ["./main"]