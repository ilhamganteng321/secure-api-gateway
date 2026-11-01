package docs

const ScalarHTML = `<!doctype html>
<html>
  <head>
    <title>Secure API Gateway - API Reference</title>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
  </head>
  <body>
    <div id="app"></div>
    <script type="module">
      import { createApiReference } from 'https://cdn.jsdelivr.net/npm/@scalar/api-reference/esm.js'
      createApiReference('#app', {
        url: '/openapi.json',
        theme: 'default',
        layout: 'modern'
      })
    </script>
  </body>
</html>`
