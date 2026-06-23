package handlers

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strings"

	"matoi/cache"
	"matoi/config"
	"matoi/models"
	"matoi/providers"

	"github.com/gofiber/fiber/v3"
	"github.com/graphql-go/graphql"
)

// GraphQLHandler manages the GraphQL schema and execution.
type GraphQLHandler struct {
	cfg                 *config.Config
	schema              graphql.Schema
	danbooruProvider    *providers.DanbooruProvider
	gelbooruProvider    *providers.GelbooruProvider
	rule34Provider      *providers.Rule34Provider
	tbibProvider        *providers.TbibProvider
	xbooruProvider      *providers.XbooruProvider
	hypnohubProvider    *providers.HypnohubProvider
	safebooruProvider   *providers.SafebooruProvider
	yandereProvider     *providers.YandereProvider
	konachanComProvider *providers.KonachanComProvider
	konachanNetProvider *providers.KonachanNetProvider
	e621Provider        *providers.E621Provider
	e926Provider        *providers.E926Provider
	furbooruProvider    *providers.FurbooruProvider
	derpibooruProvider  *providers.DerpibooruProvider
	realbooruProvider   *providers.RealbooruProvider
}

// NewGraphQLHandler initializes the GraphQL schema and returns the handler.
func NewGraphQLHandler(cfg *config.Config, dan *providers.DanbooruProvider, gel *providers.GelbooruProvider, r34 *providers.Rule34Provider, tbib *providers.TbibProvider, xbooru *providers.XbooruProvider, hypnohub *providers.HypnohubProvider, safebooru *providers.SafebooruProvider, yandere *providers.YandereProvider, konachanCom *providers.KonachanComProvider, konachanNet *providers.KonachanNetProvider, e621 *providers.E621Provider, e926 *providers.E926Provider, furbooru *providers.FurbooruProvider, derpibooru *providers.DerpibooruProvider, realbooru *providers.RealbooruProvider) *GraphQLHandler {
	h := &GraphQLHandler{
		cfg:                 cfg,
		danbooruProvider:    dan,
		gelbooruProvider:    gel,
		rule34Provider:      r34,
		tbibProvider:        tbib,
		xbooruProvider:      xbooru,
		hypnohubProvider:    hypnohub,
		safebooruProvider:   safebooru,
		yandereProvider:     yandere,
		konachanComProvider: konachanCom,
		konachanNetProvider: konachanNet,
		e621Provider:        e621,
		e926Provider:        e926,
		furbooruProvider:    furbooru,
		derpibooruProvider:  derpibooru,
		realbooruProvider:   realbooru,
	}
	h.initSchema()
	return h
}

func (h *GraphQLHandler) initSchema() {
	postType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Post",
		Fields: graphql.Fields{
			"id":                &graphql.Field{Type: graphql.Int},
			"directory":         &graphql.Field{Type: graphql.String},
			"file_url":          &graphql.Field{Type: graphql.String},
			"preview_url":       &graphql.Field{Type: graphql.String},
			"sample_url":        &graphql.Field{Type: graphql.String},
			"matoi_file_url":    &graphql.Field{Type: graphql.String},
			"matoi_preview_url": &graphql.Field{Type: graphql.String},
			"matoi_sample_url":  &graphql.Field{Type: graphql.String},
			"rating":            &graphql.Field{Type: graphql.String},
			"score":             &graphql.Field{Type: graphql.Int},
			"source":            &graphql.Field{Type: graphql.String},
			"image":             &graphql.Field{Type: graphql.String},
			"tags":              &graphql.Field{Type: graphql.NewList(graphql.String)},
			"link":              &graphql.Field{Type: graphql.String},
		},
	})

	createProviderType := func(name string, resolvePosts, resolveCompletion graphql.FieldResolveFn) *graphql.Object {
		return graphql.NewObject(graphql.ObjectConfig{
			Name: name,
			Fields: graphql.Fields{
				"posts": &graphql.Field{
					Type: graphql.NewList(postType),
					Args: graphql.FieldConfigArgument{
						"tags":    &graphql.ArgumentConfig{Type: graphql.String},
						"limit":   &graphql.ArgumentConfig{Type: graphql.Int},
						"page":    &graphql.ArgumentConfig{Type: graphql.Int},
						"shuffle": &graphql.ArgumentConfig{Type: graphql.Boolean},
					},
					Resolve: resolvePosts,
				},
				"completion": &graphql.Field{
					Type: graphql.NewList(graphql.String),
					Args: graphql.FieldConfigArgument{
						"tags": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
					},
					Resolve: resolveCompletion,
				},
			},
		})
	}

	danbooruType := createProviderType("Danbooru", h.resolveDanbooruPosts, h.resolveDanbooruCompletion)
	gelbooruType := createProviderType("Gelbooru", h.resolveGelbooruPosts, h.resolveGelbooruCompletion)
	rule34Type := createProviderType("Rule34", h.resolveRule34Posts, h.resolveRule34Completion)
	tbibType := createProviderType("Tbib", h.resolveTbibPosts, h.resolveTbibCompletion)
	xbooruType := createProviderType("Xbooru", h.resolveXbooruPosts, h.resolveXbooruCompletion)
	hypnohubType := createProviderType("Hypnohub", h.resolveHypnohubPosts, h.resolveHypnohubCompletion)
	safebooruType := createProviderType("Safebooru", h.resolveSafebooruPosts, h.resolveSafebooruCompletion)
	yandereType := createProviderType("Yandere", h.resolveYanderePosts, h.resolveYandereCompletion)
	konachanComType := createProviderType("KonachanCom", h.resolveKonachanComPosts, h.resolveKonachanComCompletion)
	konachanNetType := createProviderType("KonachanNet", h.resolveKonachanNetPosts, h.resolveKonachanNetCompletion)
	e621Type := createProviderType("E621", h.resolveE621Posts, h.resolveE621Completion)
	e926Type := createProviderType("E926", h.resolveE926Posts, h.resolveE926Completion)
	furbooruType := createProviderType("Furbooru", h.resolveFurbooruPosts, h.resolveFurbooruCompletion)
	derpibooruType := createProviderType("Derpibooru", h.resolveDerpibooruPosts, h.resolveDerpibooruCompletion)
	realbooruType := createProviderType("Realbooru", h.resolveRealbooruPosts, h.resolveRealbooruCompletion)

	noopResolve := func(_ graphql.ResolveParams) (interface{}, error) { return struct{}{}, nil }

	queryType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"danbooru":     &graphql.Field{Type: danbooruType, Resolve: noopResolve},
			"gelbooru":     &graphql.Field{Type: gelbooruType, Resolve: noopResolve},
			"rule34":       &graphql.Field{Type: rule34Type, Resolve: noopResolve},
			"tbib":         &graphql.Field{Type: tbibType, Resolve: noopResolve},
			"xbooru":       &graphql.Field{Type: xbooruType, Resolve: noopResolve},
			"hypnohub":     &graphql.Field{Type: hypnohubType, Resolve: noopResolve},
			"safebooru":    &graphql.Field{Type: safebooruType, Resolve: noopResolve},
			"yandere":      &graphql.Field{Type: yandereType, Resolve: noopResolve},
			"konachan_com": &graphql.Field{Type: konachanComType, Resolve: noopResolve},
			"konachan_net": &graphql.Field{Type: konachanNetType, Resolve: noopResolve},
			"e621":         &graphql.Field{Type: e621Type, Resolve: noopResolve},
			"e926":         &graphql.Field{Type: e926Type, Resolve: noopResolve},
			"furbooru":     &graphql.Field{Type: furbooruType, Resolve: noopResolve},
			"derpibooru":   &graphql.Field{Type: derpibooruType, Resolve: noopResolve},
			"realbooru":    &graphql.Field{Type: realbooruType, Resolve: noopResolve},
		},
	})

	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: queryType,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize GraphQL schema: %v", err))
	}

	h.schema = schema
}

type contextKey string

//nolint:gocyclo,gocognit,gocritic // Parsing request parameters makes this function complex
func (h *GraphQLHandler) resolveGenericPosts(p graphql.ResolveParams, providerName string, fetchFunc func(context.Context, string, int, int) ([]models.Post, error)) (interface{}, error) {
	tags, ok := p.Args["tags"].(string)
	if !ok {
		tags = ""
	}
	limit, ok := p.Args["limit"].(int)
	if !ok || limit <= 0 {
		limit = 20
	}
	if limit > h.cfg.PostRestReturnLimit {
		limit = h.cfg.PostRestReturnLimit
	}
	page, ok := p.Args["page"].(int)
	if !ok || page <= 0 {
		page = 1
	}
	shuffle, ok := p.Args["shuffle"].(bool)
	if !ok {
		shuffle = false
	}

	baseURL, ok := p.Context.Value(contextKey("baseURL")).(string)
	if !ok {
		baseURL = ""
	}
	ctx := p.Context

	cacheKey := fmt.Sprintf("%s:posts:%s:%d:%d", providerName, tags, limit, page)
	var posts []models.Post
	found, err := cache.Get(ctx, cacheKey, &posts)
	if err == nil && found {
		h.resolveMatoiURLs(baseURL, providerName, posts)
		if shuffle && len(posts) > 0 {
			rand.Shuffle(len(posts), func(i, j int) {
				posts[i], posts[j] = posts[j], posts[i]
			})
		}
		return posts, nil
	}

	posts, err = fetchFunc(ctx, tags, limit, page)
	if err != nil {
		return nil, fmt.Errorf("upstream fetch failed: %w", err)
	}

	if len(posts) > 0 {
		if setErr := cache.Set(ctx, cacheKey, posts, h.cfg.RedisExpireCache); setErr != nil {
			_ = setErr
		}
	}

	h.resolveMatoiURLs(baseURL, providerName, posts)
	if shuffle && len(posts) > 0 {
		rand.Shuffle(len(posts), func(i, j int) {
			posts[i], posts[j] = posts[j], posts[i]
		})
	}

	if posts == nil {
		posts = []models.Post{}
	}

	return posts, nil
}

//nolint:gocritic // signature required
func (h *GraphQLHandler) resolveGenericCompletion(p graphql.ResolveParams, fetchFunc func(context.Context, string) ([]string, error)) (interface{}, error) {
	tags, ok := p.Args["tags"].(string)
	if !ok || tags == "" {
		return []string{}, nil
	}

	ctx := p.Context
	res, err := fetchFunc(ctx, tags)
	if err != nil {
		return nil, fmt.Errorf("upstream fetch failed: %w", err)
	}

	if res == nil {
		res = []string{}
	}
	return res, nil
}

//nolint:gocritic // signature required
func (h *GraphQLHandler) resolveDanbooruPosts(p graphql.ResolveParams) (interface{}, error) {
	return h.resolveGenericPosts(p, "danbooru", h.danbooruProvider.FetchPosts)
}

//nolint:gocritic // signature required
func (h *GraphQLHandler) resolveDanbooruCompletion(p graphql.ResolveParams) (interface{}, error) {
	return h.resolveGenericCompletion(p, h.danbooruProvider.QueryCompletion)
}

//nolint:gocritic // signature required
func (h *GraphQLHandler) resolveGelbooruPosts(p graphql.ResolveParams) (interface{}, error) {
	return h.resolveGenericPosts(p, "gelbooru", h.gelbooruProvider.FetchPosts)
}

//nolint:gocritic // signature required
func (h *GraphQLHandler) resolveGelbooruCompletion(p graphql.ResolveParams) (interface{}, error) {
	return h.resolveGenericCompletion(p, h.gelbooruProvider.QueryCompletion)
}

//nolint:gocritic // signature required
func (h *GraphQLHandler) resolveRule34Posts(p graphql.ResolveParams) (interface{}, error) {
	return h.resolveGenericPosts(p, "rule34", h.rule34Provider.FetchPosts)
}

//nolint:gocritic // signature required
func (h *GraphQLHandler) resolveRule34Completion(p graphql.ResolveParams) (interface{}, error) {
	return h.resolveGenericCompletion(p, h.rule34Provider.QueryCompletion)
}

//nolint:gocritic // signature required
func (h *GraphQLHandler) resolveTbibPosts(p graphql.ResolveParams) (interface{}, error) {
	return h.resolveGenericPosts(p, "tbib", h.tbibProvider.FetchPosts)
}

//nolint:gocritic // signature required
func (h *GraphQLHandler) resolveTbibCompletion(p graphql.ResolveParams) (interface{}, error) {
	return h.resolveGenericCompletion(p, h.tbibProvider.QueryCompletion)
}

//nolint:gocritic // signature required
func (h *GraphQLHandler) resolveXbooruPosts(p graphql.ResolveParams) (interface{}, error) {
	return h.resolveGenericPosts(p, "xbooru", h.xbooruProvider.FetchPosts)
}

//nolint:gocritic // signature required
func (h *GraphQLHandler) resolveXbooruCompletion(p graphql.ResolveParams) (interface{}, error) {
	return h.resolveGenericCompletion(p, h.xbooruProvider.QueryCompletion)
}

//nolint:gocritic // signature required
func (h *GraphQLHandler) resolveHypnohubPosts(p graphql.ResolveParams) (interface{}, error) {
	return h.resolveGenericPosts(p, "hypnohub", h.hypnohubProvider.FetchPosts)
}

//nolint:gocritic // signature required
func (h *GraphQLHandler) resolveHypnohubCompletion(p graphql.ResolveParams) (interface{}, error) {
	return h.resolveGenericCompletion(p, h.hypnohubProvider.QueryCompletion)
}

func (h *GraphQLHandler) resolveMatoiURLs(baseURL, providerName string, posts []models.Post) {
	if h.cfg.ResolverURL != "" {
		baseURL = h.cfg.ResolverURL
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	for i := range posts {
		p := &posts[i]
		if p.FileURL != "" {
			p.MatoiFileURL = fmt.Sprintf("%s/api/%s/media?url=%s", baseURL, providerName, url.QueryEscape(p.FileURL))
		}
		if p.PreviewURL != "" {
			p.MatoiPreviewURL = fmt.Sprintf("%s/api/%s/media?url=%s", baseURL, providerName, url.QueryEscape(p.PreviewURL))
		}
		if p.SampleURL != "" {
			p.MatoiSampleURL = fmt.Sprintf("%s/api/%s/media?url=%s", baseURL, providerName, url.QueryEscape(p.SampleURL))
		}
	}
}

// GraphQLRequest defines the standard JSON payload for GraphQL queries.
type GraphQLRequest struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName"`
	Variables     map[string]interface{} `json:"variables"`
}

// Handle processes GraphQL requests.
//
//	@Summary	GraphQL Endpoint
//	@Tags		graphql
//	@Accept		json
//	@Produce	json
//	@Param		query	body		GraphQLRequest	true	"GraphQL Query"
//	@Success	200		{object}	map[string]interface{}
//	@Failure	400		{object}	map[string]interface{}
//	@Security	ApiKeyAuth
//	@Router		/api/graphql [post]
func (h *GraphQLHandler) Handle(c fiber.Ctx) error {
	var req GraphQLRequest
	if err := c.Bind().JSON(&req); err != nil {
		return fiber.NewError(http.StatusBadRequest, "Invalid request body")
	}

	ctx := context.WithValue(c.Context(), contextKey("baseURL"), c.BaseURL())

	result := graphql.Do(graphql.Params{
		Schema:         h.schema,
		RequestString:  req.Query,
		VariableValues: req.Variables,
		OperationName:  req.OperationName,
		Context:        ctx,
	})

	if len(result.Errors) > 0 {
		return c.Status(http.StatusBadRequest).JSON(result)
	}

	return c.Status(http.StatusOK).JSON(result)
}

// Playground serves the GraphiQL interactive UI.
//
//	@Summary	GraphiQL Playground
//	@Tags		graphql
//	@Produce	html
//	@Success	200	{string}	string	"GraphiQL HTML page"
//	@Router		/graphql [get]
func (h *GraphQLHandler) Playground(c fiber.Ctx) error {
	c.Set("Content-Type", "text/html")

	html := `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Matoi GraphiQL</title>
  <style>
    body { height: 100vh; margin: 0; width: 100%; display: flex; flex-direction: column; overflow: hidden; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; background-color: #fafafa; }
    #matoi-settings { 
      padding: 16px 24px; 
      background: #ffffff; 
      border-bottom: 1px solid #eaeaea; 
      display: flex; 
      flex-wrap: wrap;
      gap: 12px; 
      align-items: center; 
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
        width: 100% !important;
        box-sizing: border-box;
      }
    }
    #graphiql { flex: 1; }
  </style>
  <script src="https://unpkg.com/react@17/umd/react.production.min.js"></script>
  <script src="https://unpkg.com/react-dom@17/umd/react-dom.production.min.js"></script>
  <link rel="stylesheet" href="https://unpkg.com/graphiql/graphiql.min.css" />
</head>
<body>
  <div id="matoi-settings">
    <input type="text" id="api-url" placeholder="GraphQL URL" value="/api/graphql" style="width: 250px;" />
    <input type="text" id="api-key" placeholder="API Key (e.g. matoi)" value="" />
    <button onclick="updateFetcher()">Apply Settings</button>
  </div>
  <div id="graphiql">Loading...</div>
  <script src="https://unpkg.com/graphiql/graphiql.min.js"></script>
  <script>
    function renderGraphiQL() {
      const apiUrl = document.getElementById('api-url').value || '/api/graphql';
      const apiKey = document.getElementById('api-key').value;
      
      const headers = {};
      if (apiKey) {
        headers['Authorization'] = 'Bearer ' + apiKey;
      }

      const fetcher = GraphiQL.createFetcher({
        url: apiUrl,
        headers: headers
      });

      ReactDOM.render(
        React.createElement(GraphiQL, { fetcher: fetcher }),
        document.getElementById('graphiql'),
      );
    }

    function updateFetcher() {
      ReactDOM.unmountComponentAtNode(document.getElementById('graphiql'));
      renderGraphiQL();
    }

    // Initial render
    renderGraphiQL();
  </script>
</body>
</html>`

	return c.SendString(html)
}

//nolint:gocritic // signature required
func (h *GraphQLHandler) resolveSafebooruPosts(p graphql.ResolveParams) (interface{}, error) {
	return h.resolveGenericPosts(p, "safebooru", h.safebooruProvider.FetchPosts)
}

//nolint:gocritic // signature required
func (h *GraphQLHandler) resolveSafebooruCompletion(p graphql.ResolveParams) (interface{}, error) {
	return h.resolveGenericCompletion(p, h.safebooruProvider.QueryCompletion)
}

//nolint:gocritic // signature required
func (h *GraphQLHandler) resolveYanderePosts(p graphql.ResolveParams) (interface{}, error) {
	return h.resolveGenericPosts(p, "yandere", h.yandereProvider.FetchPosts)
}

//nolint:gocritic // signature required
func (h *GraphQLHandler) resolveYandereCompletion(p graphql.ResolveParams) (interface{}, error) {
	return h.resolveGenericCompletion(p, h.yandereProvider.QueryCompletion)
}

//nolint:gocritic // signature required
func (h *GraphQLHandler) resolveKonachanComPosts(p graphql.ResolveParams) (interface{}, error) {
	return h.resolveGenericPosts(p, "konachan_com", h.konachanComProvider.FetchPosts)
}

//nolint:gocritic // signature required
func (h *GraphQLHandler) resolveKonachanComCompletion(p graphql.ResolveParams) (interface{}, error) {
	return h.resolveGenericCompletion(p, h.konachanComProvider.QueryCompletion)
}

//nolint:gocritic // signature required
func (h *GraphQLHandler) resolveKonachanNetPosts(p graphql.ResolveParams) (interface{}, error) {
	return h.resolveGenericPosts(p, "konachan_net", h.konachanNetProvider.FetchPosts)
}

//nolint:gocritic // signature required
func (h *GraphQLHandler) resolveKonachanNetCompletion(p graphql.ResolveParams) (interface{}, error) {
	return h.resolveGenericCompletion(p, h.konachanNetProvider.QueryCompletion)
}

//nolint:gocritic // signature required
func (h *GraphQLHandler) resolveE621Posts(p graphql.ResolveParams) (interface{}, error) {
	return h.resolveGenericPosts(p, "e621", h.e621Provider.FetchPosts)
}

//nolint:gocritic // signature required
func (h *GraphQLHandler) resolveE621Completion(p graphql.ResolveParams) (interface{}, error) {
	return h.resolveGenericCompletion(p, h.e621Provider.QueryCompletion)
}

//nolint:gocritic // signature required
func (h *GraphQLHandler) resolveE926Posts(p graphql.ResolveParams) (interface{}, error) {
	return h.resolveGenericPosts(p, "e926", h.e926Provider.FetchPosts)
}

//nolint:gocritic // signature required
func (h *GraphQLHandler) resolveE926Completion(p graphql.ResolveParams) (interface{}, error) {
	return h.resolveGenericCompletion(p, h.e926Provider.QueryCompletion)
}

//nolint:gocritic // signature required
func (h *GraphQLHandler) resolveFurbooruPosts(p graphql.ResolveParams) (interface{}, error) {
	return h.resolveGenericPosts(p, "furbooru", h.furbooruProvider.FetchPosts)
}

//nolint:gocritic // signature required
func (h *GraphQLHandler) resolveFurbooruCompletion(p graphql.ResolveParams) (interface{}, error) {
	return h.resolveGenericCompletion(p, h.furbooruProvider.QueryCompletion)
}

//nolint:gocritic // signature required
func (h *GraphQLHandler) resolveDerpibooruPosts(p graphql.ResolveParams) (interface{}, error) {
	return h.resolveGenericPosts(p, "derpibooru", h.derpibooruProvider.FetchPosts)
}

//nolint:gocritic // signature required
func (h *GraphQLHandler) resolveDerpibooruCompletion(p graphql.ResolveParams) (interface{}, error) {
	return h.resolveGenericCompletion(p, h.derpibooruProvider.QueryCompletion)
}

//nolint:gocritic // signature required
func (h *GraphQLHandler) resolveRealbooruPosts(p graphql.ResolveParams) (interface{}, error) {
	return h.resolveGenericPosts(p, "realbooru", h.realbooruProvider.FetchPosts)
}

//nolint:gocritic // signature required
func (h *GraphQLHandler) resolveRealbooruCompletion(p graphql.ResolveParams) (interface{}, error) {
	return h.resolveGenericCompletion(p, h.realbooruProvider.QueryCompletion)
}
