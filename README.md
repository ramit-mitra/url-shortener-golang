# url-shortener-golang

## Getting Started

This is my `weekend hobby project` which is a `headless` URL shortener service.

### Tech stack

- [Go](https://go.dev/)
- [Upsun](https://upsun.com/)
- [Docker](https://www.docker.com/)

## What it does?

Generate a `short link` for a given URL.

## Routes

| HTTP Method | Path          | Description                                                        |
| ----------- | ------------- | ------------------------------------------------------------------ |
| GET         | `/`           | default endpoint                                                   |
| POST        | `/`           | create a `short link`                                              |
| GET         | `/:code`      | send `short code` and get redirected to the actual URL (if exists) |

## Development

To start the development server run:

```bash
docker compose watch
```

Open <http://localhost:1234/> with your browser to see the result. Prefer using `curl` like a true geek.

## Deploy in production

## Suggestions and feedback

Got ideas 💡 about a `feature` or an `enhancement`? Feel free to [open a PR](https://github.com/ramit-mitra/url-shortener-golang/pulls).

Found a 🐞? Feel free to [open a PR](https://github.com/ramit-mitra/url-shortener-golang/pulls) and contribute.