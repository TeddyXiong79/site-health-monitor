package main

import (
	"time"

	"github.com/gorilla/mux"
	"golang.org/x/time/rate"
)

func SetupRouter() (*mux.Router, *ipLimiter) {
	r := mux.NewRouter()

	// Dashboard
	r.HandleFunc("/", RenderDashboard).Methods("GET")

	// 内部 API（Dashboard 调用，无需认证，无限流）
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
	externalRoutes.HandleFunc("/token", APIGetToken).Methods("GET")

	// 健康检查（无需认证，无限流）
	r.HandleFunc("/api/health", APIHealth).Methods("GET")

	// 刷新节点延迟（先触发 OpenClash 延迟测试，再返回数据，无限流避免触发失败）
	r.HandleFunc("/api/refresh", APIRefresh).Methods("POST")

	return r, externalLimiter
}
