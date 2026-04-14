package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"fastlol/api"
)

func main() {
	mux := http.NewServeMux()

	// 克制查询接口
	mux.HandleFunc("/api/counter/", handleCounter)

	// Matchup 查询接口
	mux.HandleFunc("/api/matchup/", handleMatchup)

	// 健康检查
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	port := ":8080"
	log.Printf("Starting fastlol API server on %s", port)
	log.Fatal(http.ListenAndServe(port, mux))
}

func handleCounter(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 从 URL 解析 champion
	path := strings.TrimPrefix(r.URL.Path, "/api/counter/")
	parts := strings.Split(path, "/")
	if len(parts) < 1 || parts[0] == "" {
		http.Error(w, `{"error": "missing champion"}`, http.StatusBadRequest)
		return
	}
	champion := parts[0]

	// 获取 role 参数
	role := r.URL.Query().Get("role")

	// 使用 UGGScraper 获取数据
	scraper := api.NewUGGScraper()
	data, err := scraper.GetCounters(champion, role)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(data)
}

func handleMatchup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 从 URL 解析 champion 和 enemy
	path := strings.TrimPrefix(r.URL.Path, "/api/matchup/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		http.Error(w, `{"error": "missing champion or enemy"}`, http.StatusBadRequest)
		return
	}
	champion := parts[0]
	enemy := parts[1]

	// 获取 role 参数
	role := r.URL.Query().Get("role")

	// 使用 MultiScraper 获取 matchup 数据
	scraper := api.NewMultiScraper()
	matchup, err := scraper.GetMatchup(champion, enemy, role)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(matchup)
}
