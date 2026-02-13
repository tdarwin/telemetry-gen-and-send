package common

import (
	"fmt"
	"math/rand"
)

// RandomString generates a random string of the specified length
func RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// RandomInt returns a random integer between min and max (inclusive)
func RandomInt(min, max int) int {
	if min >= max {
		return min
	}
	return min + rand.Intn(max-min+1)
}

// RandomInt64 returns a random int64 between min and max (inclusive)
func RandomInt64(min, max int64) int64 {
	if min >= max {
		return min
	}
	return min + rand.Int63n(max-min+1)
}

// RandomFloat64 returns a random float64 between min and max
func RandomFloat64(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

// RandomBool returns a random boolean
func RandomBool() bool {
	return rand.Intn(2) == 1
}

// RandomChoice returns a random element from the slice
func RandomChoice[T any](choices []T) T {
	return choices[rand.Intn(len(choices))]
}

// RandomChoiceWeighted returns a random element based on weights
// weights slice must be same length as choices
func RandomChoiceWeighted[T any](choices []T, weights []int) T {
	if len(choices) != len(weights) {
		panic("choices and weights must have the same length")
	}

	totalWeight := 0
	for _, w := range weights {
		totalWeight += w
	}

	r := rand.Intn(totalWeight)
	cumulative := 0
	for i, w := range weights {
		cumulative += w
		if r < cumulative {
			return choices[i]
		}
	}

	return choices[len(choices)-1]
}

// NormalInt returns a random integer from a normal distribution
func NormalInt(mean, stdDev int) int {
	if stdDev <= 0 {
		return mean
	}
	val := rand.NormFloat64()*float64(stdDev) + float64(mean)
	result := int(val)
	if result < 1 {
		return 1 // Ensure at least 1
	}
	return result
}

// RandomDuration returns a random duration in microseconds within a range
func RandomDuration(minMicros, maxMicros int64) int64 {
	return RandomInt64(minMicros, maxMicros)
}

// RandomHTTPMethod returns a random HTTP method
func RandomHTTPMethod() string {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
	weights := []int{50, 20, 10, 5, 5, 5, 5} // GET is most common
	return RandomChoiceWeighted(methods, weights)
}

// RandomHTTPStatus returns a random HTTP status code
func RandomHTTPStatus() int {
	statuses := []int{200, 201, 204, 301, 302, 400, 401, 403, 404, 500, 502, 503}
	weights := []int{70, 5, 5, 2, 2, 3, 2, 2, 4, 2, 1, 2} // 200 is most common
	return RandomChoiceWeighted(statuses, weights)
}

// RandomHTTPPath returns a random HTTP path
func RandomHTTPPath() string {
	paths := []string{
		"/api/users",
		"/api/users/{id}",
		"/api/orders",
		"/api/orders/{id}",
		"/api/products",
		"/api/products/{id}",
		"/api/cart",
		"/api/checkout",
		"/api/search",
		"/health",
		"/metrics",
		"/",
	}
	path := RandomChoice(paths)
	// Replace {id} with random number
	if len(path) >= 4 && path[len(path)-4:] == "{id}" {
		path = path[:len(path)-4] + fmt.Sprintf("%d", RandomInt(1, 10000))
	}
	return path
}

// RandomDBSystem returns a random database system name
func RandomDBSystem() string {
	systems := []string{"postgresql", "mysql", "mongodb", "redis", "cassandra"}
	return RandomChoice(systems)
}

// RandomDBStatement returns a random database statement
func RandomDBStatement(dbSystem string) string {
	switch dbSystem {
	case "postgresql", "mysql":
		statements := []string{
			"SELECT * FROM users WHERE id = $1",
			"SELECT * FROM orders WHERE user_id = $1",
			"INSERT INTO orders (user_id, total) VALUES ($1, $2)",
			"UPDATE users SET last_login = $1 WHERE id = $2",
			"DELETE FROM cart WHERE user_id = $1",
		}
		return RandomChoice(statements)
	case "mongodb":
		statements := []string{
			"db.users.find({_id: ObjectId(...)})",
			"db.orders.find({user_id: ...})",
			"db.products.find({category: ...})",
		}
		return RandomChoice(statements)
	case "redis":
		statements := []string{
			"GET user:123",
			"SET session:abc value",
			"HGET user:123 email",
			"ZADD leaderboard 100 user:123",
		}
		return RandomChoice(statements)
	case "cassandra":
		statements := []string{
			"SELECT * FROM users WHERE id = ?",
			"INSERT INTO events (id, timestamp, data) VALUES (?, ?, ?)",
		}
		return RandomChoice(statements)
	}
	return "SELECT 1"
}

// RandomLogLevel returns a random log severity level
func RandomLogLevel() string {
	levels := []string{"DEBUG", "INFO", "WARN", "ERROR"}
	weights := []int{10, 60, 20, 10} // INFO is most common
	return RandomChoiceWeighted(levels, weights)
}

// RandomErrorType returns a random error type
func RandomErrorType() string {
	types := []string{
		"ValidationError",
		"DatabaseError",
		"NetworkError",
		"TimeoutError",
		"AuthenticationError",
		"AuthorizationError",
		"NotFoundError",
	}
	return RandomChoice(types)
}
