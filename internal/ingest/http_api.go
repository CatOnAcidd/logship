package ingest

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"sort"

	"github.com/catonacidd/logship/internal/rules"
	"github.com/catonacidd/logship/internal/store"
)

// Samples from recent logs for UI testing
func SamplesHandler(db *store.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items, err := db.Query(r.Context(), store.QueryParams{Limit: 200})
		if err != nil { http.Error(w, err.Error(), 500); return }
		out := make([]rules.EventProbe, 0, len(items))
		for _, ev := range items {
			var msg string
			if len(ev.Raw)>0 { msg = string(ev.Raw) }
			out = append(out, rules.EventProbe{
				TS: ev.TS, Source: ev.Source, Host: ev.Host, Level: ev.Level,
				Message: msg,
			})
		}
		w.Header().Set("Content-Type","application/json")
		json.NewEncoder(w).Encode(out)
	}
}

// CRUD for rules
type ruleDTO struct {
	ID int64 `json:"id"`
	Name string `json:"name"`
	Enabled bool `json:"enabled"`
	Priority int `json:"priority"`
	Source string `json:"source"`
	Action string `json:"action"`
	Predicate json.RawMessage `json:"predicate"`
}

func RulesListHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query(`SELECT id,name,enabled,priority,source,action,predicate FROM rules ORDER BY priority ASC, id ASC`)
		if err != nil { http.Error(w, err.Error(), 500); return }
		defer rows.Close()
		var list []ruleDTO
		for rows.Next() {
			var rd ruleDTO; var en int
			if err := rows.Scan(&rd.ID,&rd.Name,&en,&rd.Priority,&rd.Source,&rd.Action,&rd.Predicate); err!=nil { http.Error(w, err.Error(), 500); return }
			rd.Enabled = en==1
			list = append(list, rd)
		}
		w.Header().Set("Content-Type","application/json")
		json.NewEncoder(w).Encode(list)
	}
}

func RulesCreateHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var rd ruleDTO
		if err := json.NewDecoder(r.Body).Decode(&rd); err!=nil { http.Error(w, "bad json", 400); return }
		if rd.Name=="" || rd.Action=="" || len(rd.Predicate)==0 { http.Error(w,"missing fields",400); return }
		if rd.Priority==0 { rd.Priority = 100 }
		en := 1; if !rd.Enabled { en = 0 }
		res, err := db.Exec(`INSERT INTO rules(name,enabled,priority,source,action,predicate) VALUES(?,?,?,?,?,?)`,
			rd.Name,en,rd.Priority,rd.Source,rd.Action,string(rd.Predicate))
		if err != nil { http.Error(w, err.Error(), 500); return }
		id, _ := res.LastInsertId()
		rd.ID = id
		w.Header().Set("Content-Type","application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(rd)
	}
}

func RulesDeleteHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id=="" { http.Error(w,"id required",400); return }
		if _, err := db.Exec(`DELETE FROM rules WHERE id=?`, id); err!=nil { http.Error(w, err.Error(), 500); return }
		w.WriteHeader(204)
	}
}

// Test rule against recent samples
func RulesTestHandler(db *store.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, _ := io.ReadAll(r.Body)
		var tmp struct{
			Source string `json:"source"`
			Predicate rules.Predicate `json:"predicate"`
			Limit int `json:"limit"`
		}
		if err := json.Unmarshal(body, &tmp); err!=nil { http.Error(w,"bad json",400); return }
		items, err := db.Query(r.Context(), store.QueryParams{Limit: 200})
		if err != nil { http.Error(w, err.Error(), 500); return }
		probes := make([]rules.EventProbe, 0, len(items))
		for _, ev := range items {
			var msg string
			if len(ev.Raw)>0 { msg = string(ev.Raw) }
			probes = append(probes, rules.EventProbe{TS:ev.TS, Source: ev.Source, Host: ev.Host, Level: ev.Level, Message: msg})
		}
		engine := rules.New(db.SQL())
		_ = engine.Load(context.Background())
		idxs, err := engine.TestAgainstSamples(tmp.Predicate, tmp.Source, probes, tmp.Limit)
		if err != nil { http.Error(w, err.Error(), 400); return }
		type preview struct { Index int `json:"index"`; TS int64 `json:"ts"`; Source string `json:"source"`; Host string `json:"host"`; Message string `json:"message"` }
		out := []preview{}
		for _, i := range idxs {
			out = append(out, preview{Index:i, TS:probes[i].TS, Source:probes[i].Source, Host:probes[i].Host, Message:probes[i].Message})
		}
		sort.Slice(out, func(i,j int) bool { return out[i].TS > out[j].TS })
		json.NewEncoder(w).Encode(map[string]any{"matches": out, "count": len(out)})
	}
}

// Drops list
func DropsListHandler(db *store.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items, err := db.GetDrops(r.Context(), 100)
		if err != nil { http.Error(w, err.Error(), 500); return }
		w.Header().Set("Content-Type","application/json")
		json.NewEncoder(w).Encode(items)
	}
}

// Settings
func SettingsGetHandler(db *store.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m, err := db.GetSettings(r.Context())
		if err != nil { http.Error(w, err.Error(), 500); return }
		json.NewEncoder(w).Encode(m)
	}
}
func SettingsPutHandler(db *store.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var m map[string]string
		if err := json.NewDecoder(r.Body).Decode(&m); err!=nil { http.Error(w,"bad json",400); return }
		for k, v := range m {
			if err := db.PutSetting(r.Context(), k, v); err!=nil { http.Error(w, err.Error(), 500); return }
		}
		w.WriteHeader(204)
	}
}
