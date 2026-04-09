package main

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"golang.org/x/time/rate"
)

func SetupRouter() (*mux.Router, *ipLimiter, *ipLimiter) {
	r := mux.NewRouter()

	// Dashboard
	r.HandleFunc("/", RenderDashboard).Methods("GET")

	// 内部 API（Dashboard 同源调用，无需 Bearer Token 认证）
	r.HandleFunc("/api/data", APIGetData).Methods("GET")
	r.HandleFunc("/api/config", APISaveConfig).Methods("POST")
	r.HandleFunc("/api/test", APITestConnection).Methods("POST")

	// 外部 API（需要 Bearer Token 认证，限流）
	externalLimiter := newIPLimiter(rate.Every(100*time.Millisecond), 20)
	externalRoutes := r.PathPrefix("/api/").Subrouter()
	externalRoutes.Use(rateLimitMiddleware(externalLimiter))
	externalRoutes.HandleFunc("/summary", APIGetSummary).Methods("GET")
	externalRoutes.HandleFunc("/nodes", APIGetNodes).Methods("GET")
	externalRoutes.HandleFunc("/regions", APIGetRegions).Methods("GET")
	externalRoutes.HandleFunc("/regions/{region}/nodes", APIGetRegionNodes).Methods("GET")
	externalRoutes.HandleFunc("/nodes/filter", APIGetNodeFilter).Methods("GET")
	externalRoutes.HandleFunc("/nodes/{name}", APIGetNodeDetail).Methods("GET")

	// 健康检查（无需认证，无限流）
	r.HandleFunc("/api/health", APIHealth).Methods("GET")

	// 刷新节点延迟和切换代理（宽松限流：每 10 秒 1 次，突发 3 次）
	internalLimiter := newIPLimiter(rate.Every(10*time.Second), 3)
	r.Handle("/api/refresh", rateLimitMiddleware(internalLimiter)(http.HandlerFunc(APIRefresh))).Methods("POST")
	r.Handle("/api/switch", rateLimitMiddleware(internalLimiter)(http.HandlerFunc(APISwitchProxy))).Methods("POST")

	return r, externalLimiter, internalLimiter
}
