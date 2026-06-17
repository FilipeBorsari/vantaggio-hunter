package httputil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJSON_SetsStatusAndContentType(t *testing.T) {
	w := httptest.NewRecorder()
	JSON(w, http.StatusCreated, map[string]string{"key": "value"})

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

func TestJSON_EncodesPayload(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	w := httptest.NewRecorder()
	JSON(w, http.StatusOK, payload{Name: "test", Age: 42})

	var got payload
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Name != "test" || got.Age != 42 {
		t.Errorf("body = %+v, want {Name:test Age:42}", got)
	}
}

func TestJSON_SlicePayload(t *testing.T) {
	w := httptest.NewRecorder()
	JSON(w, http.StatusOK, []string{"a", "b", "c"})

	var got []string
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 3 || got[0] != "a" {
		t.Errorf("body = %v, want [a b c]", got)
	}
}

func TestError_SetsStatusAndMessage(t *testing.T) {
	w := httptest.NewRecorder()
	Error(w, http.StatusBadRequest, "entrada inválida")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["error"] != "entrada inválida" {
		t.Errorf("error = %q, want entrada inválida", body["error"])
	}
}

func TestError_ContentType(t *testing.T) {
	w := httptest.NewRecorder()
	Error(w, http.StatusUnauthorized, "não autorizado")
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

func TestError_Codes(t *testing.T) {
	cases := []struct {
		code int
		msg  string
	}{
		{http.StatusBadRequest, "bad request"},
		{http.StatusUnauthorized, "unauthorized"},
		{http.StatusForbidden, "forbidden"},
		{http.StatusNotFound, "not found"},
		{http.StatusConflict, "conflict"},
		{http.StatusInternalServerError, "internal error"},
	}
	for _, tc := range cases {
		w := httptest.NewRecorder()
		Error(w, tc.code, tc.msg)
		if w.Code != tc.code {
			t.Errorf("code %d: got %d", tc.code, w.Code)
		}
	}
}
