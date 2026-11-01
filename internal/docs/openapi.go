package docs

// OpenAPISpec is a minimal OpenAPI 3.1 description for the Secure API Gateway.
// It is intentionally small and can be expanded as endpoints evolve.
const OpenAPISpec = `{
  "openapi": "3.1.0",
  "info": {
    "title": "Secure API Gateway",
    "version": "1.0.0",
    "description": "Authentication, authorization, rate limiting, and audit logging API."
  },
  "servers": [
    {
      "url": "http://localhost:3000"
    }
  ],
  "paths": {
    "/health/live": {
      "get": {
        "summary": "Liveness check",
        "responses": {
          "204": {
            "description": "Application process is alive"
          }
        }
      }
    },
    "/health/ready": {
      "get": {
        "summary": "Readiness check",
        "responses": {
          "200": {
            "description": "Application and dependencies are ready"
          },
          "503": {
            "description": "Application is not ready"
          }
        }
      }
    },
    "/api/v1/auth/register": {
      "post": {
        "summary": "Register a new user",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["email", "password"],
                "properties": {
                  "email": { "type": "string", "format": "email" },
                  "password": { "type": "string" }
                }
              }
            }
          }
        },
        "responses": {
          "201": { "description": "Registration successful" },
          "400": { "description": "Invalid input" },
          "409": { "description": "Email already registered" }
        }
      }
    },
    "/api/v1/auth/login": {
      "post": {
        "summary": "Login and receive access and refresh tokens",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["email", "password"],
                "properties": {
                  "email": { "type": "string", "format": "email" },
                  "password": { "type": "string" }
                }
              }
            }
          }
        },
        "responses": {
          "200": { "description": "Login successful" },
          "401": { "description": "Invalid credentials" },
          "403": { "description": "Account inactive" },
          "429": { "description": "Rate limit exceeded" }
        }
      }
    },
    "/api/v1/auth/refresh": {
      "post": {
        "summary": "Rotate refresh token",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["refresh_token"],
                "properties": {
                  "refresh_token": { "type": "string" }
                }
              }
            }
          }
        },
        "responses": {
          "200": { "description": "Tokens rotated" },
          "401": { "description": "Invalid or reused refresh token" }
        }
      }
    },
    "/api/v1/auth/logout": {
      "post": {
        "summary": "Revoke refresh token",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["refresh_token"],
                "properties": {
                  "refresh_token": { "type": "string" }
                }
              }
            }
          }
        },
        "responses": {
          "200": { "description": "Logout successful" },
          "401": { "description": "Invalid refresh token" }
        }
      }
    },
    "/api/v1/me": {
      "get": {
        "summary": "Get current user id",
        "security": [{ "bearerAuth": [] }],
        "responses": {
          "200": { "description": "Authenticated user" },
          "401": { "description": "Authentication required" }
        }
      }
    },
    "/api/v1/admin": {
      "get": {
        "summary": "Admin-only endpoint",
        "security": [{ "bearerAuth": [] }],
        "responses": {
          "200": { "description": "Admin access granted" },
          "401": { "description": "Authentication required" },
          "403": { "description": "Insufficient permissions" }
        }
      }
    }
  },
  "components": {
    "securitySchemes": {
      "bearerAuth": {
        "type": "http",
        "scheme": "bearer",
        "bearerFormat": "JWT"
      }
    }
  }
}`
