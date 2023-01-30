FROM golang:alpine AS build
WORKDIR /app
# /app/quizzer
COPY quizzer . 
RUN go build -o quizzer .

FROM alpine
WORKDIR /app
COPY --from=build /app/ /app/
ENTRYPOINT ["/app/quizzer"]

# The above Dockerfile works perfectly fine and also bundles the Cobra CLI quizzer tool. But running the cli with docker container creates a problem, as running the cli as a container is not user interactive friendly. The user input scanner doesn't work at all.