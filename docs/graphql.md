# Matoi GraphQL Usage Guide

GraphQL support is available as a single unified endpoint. It allows you to fetch exact fields you need without over-fetching or under-fetching, and supports querying both `posts` and `completion` simultaneously.

**Endpoint:** `POST /api/graphql`
**Headers Required:**
- `Authorization: Bearer <API_KEY>` (e.g., `matoi`)
- `Content-Type: application/json`

---

## 1. Schema Structure

Currently supported providers: **Rule34**.
The structure uses a nested object approach.

### Query Example (All-in-one)
You can fetch both posts and tags completion in a single request:

```graphql
query {
  rule34 {
    posts(tags: "yuri", limit: 5) {
      id
      file_url
      matoi_file_url
      rating
      tags
    }
    completion(tags: "jeanne")
  }
}
```

---

## 2. Interactive Documentation (Swagger vs GraphiQL)

Matoi provides two different UI playgrounds for testing endpoints. It is **highly recommended** to use GraphiQL for testing GraphQL queries.

### Using GraphiQL (`/graphql`)
GraphiQL natively understands GraphQL. You can write **pure** GraphQL syntax directly into the editor without any JSON wrapping.
```graphql
query {
  rule34 {
    posts(tags: "yuri", limit: 5) {
      id
      rating
      file_url
      tags
    }
  }
}
```

### Using Swagger UI (`/swagger` or `docs/index.html`)
Swagger UI treats `/api/graphql` as a standard REST endpoint. Because the `Content-Type` is `application/json`, you **must** wrap your GraphQL query inside a valid JSON object. You **cannot** paste pure GraphQL with unescaped newlines here.

**Correct JSON Format for Swagger:**
```json
{
  "query": "query { rule34 { posts(tags: \"yuri\", limit: 5) { id rating file_url tags } } }"
}
```

---

## 3. cURL Usage

> **Note for Windows Users:** When using `curl.exe` in PowerShell or CMD, ensure you properly escape double quotes (`\"`) inside the JSON payload.

### Fetch Posts Only
```bash
curl -X POST http://localhost:3000/api/graphql \
  -H "Authorization: Bearer matoi" \
  -H "Content-Type: application/json" \
  -d "{\"query\": \"query { rule34 { posts(tags: \\\"yuri\\\", limit: 5) { id file_url matoi_file_url } } }\"}"
```

### Fetch Completion Only
```bash
curl -X POST http://localhost:3000/api/graphql \
  -H "Authorization: Bearer matoi" \
  -H "Content-Type: application/json" \
  -d "{\"query\": \"query { rule34 { completion(tags: \\\"jeanne\\\") } }\"}"
```

---

## 4. Node.js Usage

The most robust way to use GraphQL in Node.js/Frontend is by separating the `query` string and `variables` object.

```javascript
async function fetchMatoiGraphQL() {
  const query = `
    query GetRule34Data($tags: String!, $limit: Int, $completionTag: String!) {
      rule34 {
        posts(tags: $tags, limit: $limit) {
          id
          file_url
          matoi_file_url
          rating
        }
        completion(tags: $completionTag)
      }
    }
  `;

  const variables = {
    tags: "yuri",
    limit: 5,
    completionTag: "jeanne"
  };

  try {
    const response = await fetch("http://localhost:3000/api/graphql", {
      method: "POST",
      headers: {
        "Authorization": "Bearer matoi",
        "Content-Type": "application/json"
      },
      body: JSON.stringify({
        query: query,
        variables: variables
      })
    });

    const data = await response.json();
    console.log(JSON.stringify(data, null, 2));
  } catch (error) {
    console.error("GraphQL Error:", error);
  }
}

fetchMatoiGraphQL();
```

---

---

## 5. Bun (ESM) Usage

Bun supports modern ES modules natively. You can use top-level `await` and the native `fetch` API directly.

```javascript
// index.ts or index.mjs
const query = `
  query GetRule34Data($tags: String!, $limit: Int) {
    rule34 {
      posts(tags: $tags, limit: $limit) {
        id
        file_url
        rating
      }
    }
  }
`;

const response = await fetch("http://localhost:3000/api/graphql", {
  method: "POST",
  headers: {
    "Authorization": "Bearer matoi",
    "Content-Type": "application/json"
  },
  body: JSON.stringify({
    query,
    variables: { tags: "yuri", limit: 5 }
  })
});

const data = await response.json();
console.log(JSON.stringify(data, null, 2));
```

---

## 6. Go Usage

Calling the GraphQL endpoint natively using Go's standard `net/http` library.

```go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func main() {
	query := `
		query GetRule34Data($tags: String!, $limit: Int) {
			rule34 {
				posts(tags: $tags, limit: $limit) {
					id
					file_url
					matoi_file_url
				}
			}
		}
	`

	payload := map[string]interface{}{
		"query": query,
		"variables": map[string]interface{}{
			"tags":  "yuri",
			"limit": 5,
		},
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "http://localhost:3000/api/graphql", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer matoi")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	fmt.Println(string(respBody))
}
```

---

## 7. Troubleshooting
- **Missing Matoi URLs (`matoi_file_url`)?** Remember that GraphQL only returns the fields you explicitly request. Ensure you add `matoi_file_url` into your query block.
- **Syntax Errors in cURL?** Ensure you are using `curl.exe` and properly escaping JSON strings if you are on Windows. Using tools like Postman, Insomnia, or ThunderClient with a raw JSON body is highly recommended.
