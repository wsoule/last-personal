# Microservices

## What is Microservices?

Microservices is a software architecture style that structures an application as a collection of small services, which can be developed, deployed, and scaled independently.

## Why Microservices?

- Scalability
- Agility
- Resilience
- Security
- Maintainability
- Portability
- Testability
- Reusability

## How to build Microservices?

idk, thats why i made one.

[https://github.com/wsoule/infrastructure](https://github.com/wsoule/infrastructure)

It is made using Golang, docker, docker compose, and html to view the services. It is hosted on railway.

[https://infra.wyat.me](https://infra.wyat.me)

I just made 2 microservices: users and products.
Then i have a api-gateway that routes the requests to the correct service.

It was a cool project, very small - i learned about gateways, reverse proxy, internal networks, a little about goroutines (for startup and gentle shutdown of each service).
