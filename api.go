package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
)

type Server struct {
	port  string
	store *Store
}

type APIFunc func(w http.ResponseWriter, r *http.Request) error

func makeHandler(handler APIFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Request from %s\t%s %s\n", r.RemoteAddr, r.Method, r.URL.Path)

		err := handler(w, r)
		if err != nil {
			if e, ok := err.(APIError); ok {
				log.Println("API error:", e.Msg)
				writeJSON(w, e.StatusCode, e)
			} else {
				log.Println("error:", err)
				writeJSON(w, http.StatusInternalServerError, "Internal Error")
			}
		}
	}
}

func getPathId(wildcard string, r *http.Request) (int, error) {
	v := r.PathValue(wildcard)
	if v == "" {
		return 0, errors.New("unable to get path id")
	}

	id, err := strconv.Atoi(v)
	return id, err
}

func writeJSON(w http.ResponseWriter, status int, data any) error {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}

func NewServer(port string, store *Store) *Server {
	s := &Server{
		port: port,
		store: store,
	}

	http.HandleFunc("GET /api/components", makeHandler(s.getComponents))
	http.HandleFunc("GET /api/components/{id}", makeHandler(s.getComponentById))
	http.HandleFunc("POST /api/components", makeHandler(s.createComponent))

	http.HandleFunc("GET /api/assemblies", makeHandler(s.getAssemblies))
	http.HandleFunc("GET /api/assemblies/{id}", makeHandler(s.getAssemblyById))
	http.HandleFunc("POST /api/assemblies", makeHandler(s.createAssembly))
	http.HandleFunc("POST /api/assemblies/{id}/versions", makeHandler(s.createAssemblyVersion))
	http.HandleFunc("GET /api/assemblies/{id}/versions", makeHandler(s.getAssemblyVersions))

	http.HandleFunc("GET /api/versions/{id}", makeHandler(s.getAssemblyVersionById))
	http.HandleFunc("GET /api/versions/{id}/bom", makeHandler(s.getBillOfMaterials))
	http.HandleFunc("POST /api/versions/{id}/publish", makeHandler(s.publishVersion))
	http.HandleFunc("POST /api/versions/{id}/deprecate", makeHandler(s.deprecateVersion))

	return s
}

func (s *Server) Run() {
	log.Println("Server running on", s.port)
	err := http.ListenAndServe(s.port, nil)
	log.Fatal(err)
}

func (s *Server) getComponentById(w http.ResponseWriter, r *http.Request) error {
	id, err := getPathId("id", r)
	if err != nil {
		return BadRequest()
	}

	result, err := getComponentById(s.store.db, id)
	if err != nil {
		return err
	}

	return writeJSON(w, http.StatusOK, result)
}

func (s *Server) getComponents(w http.ResponseWriter, r *http.Request) error {
	items, err := getComponents(s.store.db)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, items)
}

func (s *Server) createComponent(w http.ResponseWriter, r *http.Request) error {
	var req CreateComponentParams
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return BadRequest()
	}

	result, err := createComponent(s.store.db, req)
	if err != nil {
		return err
	}

	return writeJSON(w, http.StatusOK, result)
}

func (s *Server) getAssemblyById(w http.ResponseWriter, r *http.Request) error {
	id, err := getPathId("id", r)
	if err != nil {
		return BadRequest()
	}

	asm, err := getAssemblyById(s.store.db, id)
	if err != nil {
		return err
	}

	return writeJSON(w, http.StatusOK, asm)
}

func (s *Server) getAssemblies(w http.ResponseWriter, r *http.Request) error {
	assemblies, err := getAssemblies(s.store.db)
	if err != nil {
		return err
	}

	return writeJSON(w, http.StatusOK, assemblies)
}

func (s *Server) createAssembly(w http.ResponseWriter, r *http.Request) error {
	var req CreateAssemblyParams
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return BadRequest()
	}

	result, err := createAssembly(s.store.db, req)
	if err != nil {
		return err
	}

	return writeJSON(w, http.StatusOK, result)
}

func (s *Server) createAssemblyVersion(w http.ResponseWriter, r *http.Request) error {
	itemId, err := getPathId("id", r)
	if err != nil {
		return BadRequest()
	}

	var req CreateItemVersionParams
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return BadRequest()
	}

	itemVersion, err := createItemVersionWrapper(s.store.db, itemId, req)
	if err != nil {
		return err
	}

	return writeJSON(w, http.StatusOK, itemVersion)
}

func (s *Server) getAssemblyVersionById(w http.ResponseWriter, r *http.Request) error {
	versionId, err := getPathId("id", r)
	if err != nil {
		return BadRequest()
	}

	version, err := getItemVersionById(s.store.db, versionId)
	if err != nil {
		return err
	}

	return writeJSON(w, http.StatusOK, version)
}

func (s *Server) getAssemblyVersions(w http.ResponseWriter, r *http.Request) error {
	return NotImplemented()
}

func (s *Server) getBillOfMaterials(w http.ResponseWriter, r *http.Request) error {
	versionId, err := getPathId("id", r)
	if err != nil {
		return BadRequest()
	}

	bom, err := getBom(s.store.db, versionId)
	if err != nil {
		return err
	}

	return writeJSON(w, http.StatusOK, bom)
}

func (s *Server) publishVersion(w http.ResponseWriter, r *http.Request) error {
	versionId, err := getPathId("id", r)
	if err != nil {
		return BadRequest()
	}

	err = publishVersion(s.store.db, versionId)
	if err != nil {
		return err
	}

	return writeJSON(w, http.StatusOK, nil)
}

func (s *Server) deprecateVersion(w http.ResponseWriter, r *http.Request) error {
	versionId, err := getPathId("id", r)
	if err != nil {
		return BadRequest()
	}

	err = publishVersion(s.store.db, versionId)
	if err != nil {
		return err
	}

	return writeJSON(w, http.StatusOK, nil)
}
