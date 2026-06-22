package dev_tools

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateDocsUI(t *testing.T) {
	htmlTemplate := `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Matoi API Documentation</title>
  <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui.css" />
  <style>
    body { margin: 0; padding: 0; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; background-color: #fafafa; }
    #matoi-settings { 
      padding: 16px 24px; 
      background: #ffffff; 
      border-bottom: 1px solid #eaeaea; 
      display: flex; 
      flex-wrap: wrap;
      gap: 12px; 
      align-items: center; 
      justify-content: center;
      box-shadow: 0 1px 3px rgba(0,0,0,0.04);
    }
    #matoi-settings strong {
      font-size: 15px;
      color: #333;
      margin-right: 8px;
    }
    #matoi-settings input { 
      padding: 8px 12px; 
      border: 1px solid #e1e4e8; 
      border-radius: 6px; 
      font-size: 14px; 
      outline: none;
      transition: all 0.2s ease;
      background-color: #f6f8fa;
    }
    #matoi-settings input:focus {
      border-color: #0366d6;
      box-shadow: 0 0 0 3px rgba(3, 102, 214, 0.3);
      background-color: #fff;
    }
    #matoi-settings button { 
      padding: 8px 16px; 
      background: #2ea44f; 
      color: white; 
      border: 1px solid rgba(27, 31, 35, 0.15); 
      border-radius: 6px; 
      cursor: pointer; 
      font-size: 14px; 
      font-weight: 500;
      transition: all 0.2s ease;
    }
    #matoi-settings button:hover { 
      background: #2c974b; 
    }
    #matoi-settings button:active {
      background: #298e46;
      box-shadow: inset 0 1px 0 rgba(22, 38, 43, 0.2);
    }
    @media (max-width: 600px) {
      #matoi-settings {
        flex-direction: column;
        align-items: stretch;
      }
      #matoi-settings input, #matoi-settings button {
        width: 100%% !important;
        box-sizing: border-box;
      }
    }
    .swagger-ui .topbar { display: none; } /* Hide default topbar */
  </style>
</head>
<body>
  <div id="matoi-settings">
    <input type="text" id="api-url" placeholder="e.g. http://localhost:3000" value="http://localhost:3000" style="width: 250px;" />
    <input type="text" id="api-key" placeholder="API Key" value="" />
    <button onclick="renderSwagger()">Apply Settings</button>
  </div>
  
  <div id="swagger-ui"></div>

  <script src="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui-bundle.js"></script>
  <script src="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui-standalone-preset.js"></script>
  <script>
    const specData = %s;

    function renderSwagger() {
      if (!specData) return;

      const apiUrl = document.getElementById('api-url').value;
      let urlObj;
      try {
        urlObj = new URL(apiUrl);
      } catch (err) {
        alert("Invalid URL format. Please include http:// or https://");
        return;
      }
      
      // Create a deep copy so we don't modify the original object incorrectly
      const spec = JSON.parse(JSON.stringify(specData));
      
      // Override the host and schemes from the input URL
      spec.host = urlObj.host;
      spec.schemes = [urlObj.protocol.replace(':', '')];
      spec.basePath = urlObj.pathname === '/' ? '' : urlObj.pathname;

      window.ui = SwaggerUIBundle({
        spec: spec,
        dom_id: '#swagger-ui',
        deepLinking: true,
        presets: [
          SwaggerUIBundle.presets.apis,
          SwaggerUIStandalonePreset
        ],
        plugins: [
          SwaggerUIBundle.plugins.DownloadUrl
        ],
        layout: "StandaloneLayout",
        requestInterceptor: function(req) {
          const key = document.getElementById('api-key').value;
          if (key) {
            req.headers['Authorization'] = 'Bearer ' + key;
          }
          return req;
        }
      });
    }

    window.onload = function() {
      renderSwagger();
    };
  </script>
</body>
</html>`

	// Determine the correct path to the docs directory from dev_tools
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}

	docsDir := filepath.Join(cwd, "..", "docs")
	swaggerJSONPath := filepath.Join(docsDir, "swagger.json")
	indexPath := filepath.Join(docsDir, "index.html")

	// Read swagger.json
	swaggerData, err := os.ReadFile(swaggerJSONPath)
	if err != nil {
		t.Fatalf("Failed to read swagger.json (make sure you ran 'swag init' first): %v", err)
	}

	// Inject swagger.json directly into the HTML
	finalHTML := fmt.Sprintf(htmlTemplate, string(swaggerData))

	// Ensure docs directory exists
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatalf("Failed to create docs directory: %v", err)
	}

	// Write the HTML content to docs/index.html
	if err := os.WriteFile(indexPath, []byte(finalHTML), 0o644); err != nil {
		t.Fatalf("Failed to write docs/index.html: %v", err)
	}

	t.Logf("Successfully generated %s", indexPath)
}
