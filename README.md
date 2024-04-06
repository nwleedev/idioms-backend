# Idioms Backend

`useidioms.com` Backend with Go

### Dependencies

- sqlc
- sqlx, squirrel
- chi router
- aws-sdk-go-v2

### Commands

- `go run .`

Run dev backend with port 8081

- `go build -o build/app`

Build Go backend into the folder `build`

### API Routes

`/idioms`

- Fetch idioms with thumbnail
- Query Parameters
  - orderBy
    - created_at
    - idiom
  - orderDirection
    - asc
    - desc
  - count
  - nextToken
  - prevToken

`/idioms/{id}`

- Fetch a idiom by id

`/idioms/{id}/related`

- Fetch idioms near at the idiom by id

`/idioms/search`

- Fetch idioms by keywords

#### API Routes for admin

`/idioms/inputs`

- Create idioms by input

```JSON
[{
  "idiom": "string",
  "meaning": "string"
}]
```

`/idioms/thumbnail/draft`

- Create thumbnail draft

`/idioms/thumbnail/file`

- Upload idiom thumbnail with file

`/idioms/thumbnail/url`

- Upload idiom thumbnail with url

`/idioms/{id}/thumbnail`

- Update thumbnail prompt by id
