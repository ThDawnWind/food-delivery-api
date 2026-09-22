package openapi

import (
	_ "embed"
	"net/http"
)

//go:embed openapi.yaml
var spec []byte

const scalarHTML = `<!doctype html>
<html>
<head>
	<title>Food Delivery API</title>
	<meta charset="utf-8" />
	<meta name="viewport" content="width=device-width, initial-scale=1" />
</head>

<body>
	<div id="app"></div>

	<script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference@1.70.0"></script>

	<script>
		Scalar.createApiReference('#app', {
			url: '/openapi.yaml',
			theme: 'default'
		})
	</script>
</body>
</html>`

func SpecHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	_, _ = w.Write(spec)
}

func DocsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	_, _ = w.Write([]byte(scalarHTML))
}
