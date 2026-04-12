# Configuration

## Config

```go
type Config struct {
	URL       string
	Username  string
	Password  string
	Namespace *string
	Database  *string
	MaxConns  int
	Timeout   time.Duration
}
```

## Notes

- `Timeout` defaults to `30s`
- `MaxConns` is kept for future work

## Options

```go
db, err := surrealgoorm.Connect(ctx, surrealgoorm.Config{
	URL:      "ws://localhost:8000",
	Username: "root",
	Password: "root",
},
	surrealgoorm.WithTimeout(60*time.Second),
)
```

## DB Methods

### Raw

```go
raw := db.Raw()
```

### Exec

```go
err := db.Exec(ctx, "DELETE FROM users WHERE id = $id", map[string]any{
	"id": "users:john",
})
```

### Transaction

```go
err := db.Transaction(ctx, func(tx *surrealgoorm.DB) error {
	return tx.Exec(ctx, "CREATE users:john SET name = $name", map[string]any{
		"name": "John",
	})
})
```
