package bootstrap

import (
	"context"
	"fmt"
	"hypefast-api/lib/utils"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// CustomClaims JWT custom claims
type CustomClaims struct {
	MemberCode string `json:"member_code"`
	jwt.StandardClaims
}

// RegisterClaims JWT custom claims
type RegisterClaims struct {
	Token string `json:"token_register"`
	jwt.StandardClaims
}

var (
	mustHeader = []string{"X-Channel", "Content-Type"}
	headerVal  = []string{"webtraveller", "webcms", "application/json"}
)

func userContext(ctx context.Context, subject, id interface{}) context.Context {
	return context.WithValue(ctx, subject, id)
}

const pingReqURI = "/v1/ping"

func isPingRequest(r *http.Request) bool {
	return r.RequestURI == pingReqURI
}

// NotfoundMiddleware A custom not found response.
func (app *App) NotfoundMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tctx := chi.NewRouteContext()
		rctx := chi.RouteContext(r.Context())

		if !rctx.Routes.Match(tctx, r.Method, r.URL.Path) {
			app.SendNotfound(w, "Sorry. We couldn't find that page")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Recoverer ...
func (app *App) Recoverer(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil {
				logEntry := middleware.GetLogEntry(r)
				if logEntry != nil {
					logEntry.Panic(rvr, debug.Stack())
				} else {
					debug.PrintStack()
				}

				app.Log.FromDefault().WithFields(logrus.Fields{
					"Panic": rvr,
				}).Errorf("Panic: %v \n %v", rvr, string(debug.Stack()))

				app.SendBadRequest(w, "Something error with our system. Please contact our administrator")
				return
			}
		}()

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

// VerifyJwtToken ...
func (app *App) VerifyJwtToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := &CustomClaims{}
		tokenAuth := r.Header.Get("Authorization")
		_, err := jwt.ParseWithClaims(tokenAuth, claims, func(token *jwt.Token) (interface{}, error) {
			if jwt.SigningMethodHS256 != token.Method {
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}

			secret := app.Config.GetString("app.key")
			return []byte(secret), nil
		})

		if err != nil {
			msg := "token is invalid"
			if mErr, ok := err.(*jwt.ValidationError); ok {
				if mErr.Errors == jwt.ValidationErrorExpired {
					msg = "token is expired"
				}
			}

			app.SendAuthError(w, msg)
			return
		}

		// TODO: should check to redis/db is token expired or not

		ctx := userContext(r.Context(), "mcode", fmt.Sprint(claims.MemberCode))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// VerifyJwtTokenRegister ...
func (app *App) VerifyJwtTokenRegister(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := &RegisterClaims{}
		tokenAuth := r.Header.Get("Authorization")
		_, err := jwt.ParseWithClaims(tokenAuth, claims, func(token *jwt.Token) (interface{}, error) {
			if jwt.SigningMethodHS256 != token.Method {
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}

			secret := app.Config.GetString("app.key")
			return []byte(secret), nil
		})

		if err != nil {
			msg := "token is invalid"
			if mErr, ok := err.(*jwt.ValidationError); ok {
				if mErr.Errors == jwt.ValidationErrorExpired {
					msg = "token is expired"
				}
			}

			app.SendAuthError(w, msg)
			return
		}

		ctx := userContext(r.Context(), "rcode", fmt.Sprint(claims.Token))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// HeaderCheckerMiddleware check the necesarry headers
func (app *App) HeaderCheckerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, v := range mustHeader {
			if len(r.Header.Get(v)) == 0 || !utils.Contains(headerVal, r.Header.Get(v)) {
				app.SendBadRequest(w, fmt.Sprintf("undefined %s header or wrong value of header", v))
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// promotheus section
// https://github.com/766b/chi-prometheus/blob/master/middleware.go

var (
	dflBuckets = []float64{300, 1200, 5000}
)

const (
	reqsName           = "chi_requests_total"
	latencyName        = "chi_request_duration_milliseconds"
	patternReqsName    = "chi_pattern_requests_total"
	patternLatencyName = "chi_pattern_request_duration_milliseconds"
)

// Middleware is a handler that exposes prometheus metrics for the number of requests,
// the latency and the response size, partitioned by status code, method and HTTP path.
type PromotMiddleware struct {
	reqs    *prometheus.CounterVec
	latency *prometheus.HistogramVec
}

// NewPrometMiddleware returns a new prometheus Middleware handler.
func (app *App) NewPrometMiddleware(name string, buckets ...float64) func(next http.Handler) http.Handler {
	var m PromotMiddleware
	m.reqs = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:        reqsName,
			Help:        "How many HTTP requests processed, partitioned by status code, method and HTTP path.",
			ConstLabels: prometheus.Labels{"service": name},
		},
		[]string{"code", "method", "path", "channel"},
	)
	prometheus.MustRegister(m.reqs)

	if len(buckets) == 0 {
		buckets = dflBuckets
	}
	m.latency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:        latencyName,
		Help:        "How long it took to process the request, partitioned by status code, method and HTTP path.",
		ConstLabels: prometheus.Labels{"service": name},
		Buckets:     buckets,
	},
		[]string{"code", "method", "path", "channel"},
	)
	prometheus.MustRegister(m.latency)
	return m.handler
}

func (c PromotMiddleware) handler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		c.reqs.WithLabelValues(http.StatusText(ww.Status()), r.Method, r.URL.Path, r.Header.Get("X-CHANNEL")).Inc()
		c.latency.WithLabelValues(http.StatusText(ww.Status()), r.Method, r.URL.Path, r.Header.Get("X-CHANNEL")).Observe(float64(time.Since(start).Nanoseconds()) / 1000000)
	}
	return http.HandlerFunc(fn)
}

// NewPrometPatternMiddleware returns a new prometheus Middleware handler that groups requests by the chi routing pattern.
// https://github.com/edjumacator/chi-prometheus/blob/add-route-pattern-support/pattern_example/main.go
// EX: /users/{firstName} instead of /users/bob
func (app *App) NewPrometPatternMiddleware(name string, buckets ...float64) func(next http.Handler) http.Handler {
	var m PromotMiddleware
	m.reqs = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:        patternReqsName,
			Help:        "How many HTTP requests processed, partitioned by status code, method and HTTP path (with patterns).",
			ConstLabels: prometheus.Labels{"service": name},
		},
		[]string{"code", "method", "path", "channel"},
	)
	prometheus.MustRegister(m.reqs)

	if len(buckets) == 0 {
		buckets = dflBuckets
	}
	m.latency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:        patternLatencyName,
		Help:        "How long it took to process the request, partitioned by status code, method and HTTP path (with patterns).",
		ConstLabels: prometheus.Labels{"service": name},
		Buckets:     buckets,
	},
		[]string{"code", "method", "path", "channel"},
	)
	prometheus.MustRegister(m.latency)
	return m.patternHandler
}

func (c PromotMiddleware) patternHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		rctx := chi.RouteContext(r.Context())
		routePattern := strings.Join(rctx.RoutePatterns, "")
		routePattern = strings.Replace(routePattern, "/*/", "/", -1)

		// dt := time.Now().In(time.UTC).Format("2006-01-02 15:04:05")

		c.reqs.WithLabelValues(http.StatusText(ww.Status()), r.Method, routePattern, r.Header.Get("X-CHANNEL")).Inc()
		c.latency.WithLabelValues(http.StatusText(ww.Status()), r.Method, routePattern, r.Header.Get("X-CHANNEL")).Observe(float64(time.Since(start).Nanoseconds()) / 1000000)
	}
	return http.HandlerFunc(fn)
}
