package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
)

type Server struct {
	port  string
	store *Store
}

func NewServer(port string, store *Store) *Server {
	s := &Server{
		port: port,
		store: store,
	}

	s.Handle("GET /api/items", s.getItems)
	s.Handle("GET /api/items/{id}", s.getItemById)
	s.Handle("GET /api/items/{id}/thumbnail", s.getItemThumbnail)
	s.Handle("POST /api/items/{id}/thumbnail", s.uploadItemThumbnail)

	s.Handle("GET /api/components", s.getComponents)
	s.Handle("GET /api/components/{id}", s.getComponentById)
	s.Handle("POST /api/components", s.createComponent)

	s.Handle("GET /api/assemblies", s.getAssemblies)
	s.Handle("GET /api/assemblies/{id}", s.getAssemblyById)
	s.Handle("POST /api/assemblies", s.createAssembly)
	s.Handle("POST /api/assemblies/{id}/versions", s.createAssemblyVersion)
	s.Handle("GET /api/assemblies/{id}/versions", s.getAssemblyVersions)

	s.Handle("GET /api/versions/{id}", s.getAssemblyVersionById)
	s.Handle("GET /api/versions/{id}/bom", s.getBillOfMaterials)
	s.Handle("POST /api/versions/{id}/publish", s.publishVersion)
	s.Handle("POST /api/versions/{id}/deprecate", s.deprecateVersion)

	s.Handle("GET /api/images/{id}", s.getImageById)

	return s
}

func (s *Server) Run() {
	log.Println("Server running on", s.port)
	err := http.ListenAndServe(s.port, nil)
	log.Fatal(err)
}

func (s *Server) Handle(pattern string, handler APIFunc) {
	http.HandleFunc(pattern, makeHandler(handler))
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

func writeImage(w http.ResponseWriter, img Image, content io.ReadCloser) error {
    w.Header().Set("Content-Type", img.MimeType)
    w.Header().Set("Content-Length", strconv.Itoa(img.SizeBytes))

	_, err := io.Copy(w, content)
	return err
}

func (s *Server) getItems(w http.ResponseWriter, r *http.Request) error {
	items, err := getItems(s.store.db)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, items)
}

func (s *Server) getItemById(w http.ResponseWriter, r *http.Request) error {
	id, err := getPathId("id", r)
	if err != nil {
		return BadRequest()
	}

	result, err := getItemById(s.store.db, id)
	if err != nil {
		return err
	}

	return writeJSON(w, http.StatusOK, result)
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
	itemId, err := getPathId("id", r)
	if err != nil {
		return BadRequest()
	}

	versions, err := getItemVersionsByItem(s.store.db, itemId)
	if err != nil {
		return err
	}

	return writeJSON(w, http.StatusOK, versions)
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

func (s *Server) getItemThumbnail(w http.ResponseWriter, r *http.Request) error {
	itemId, err := getPathId("id", r)
	if err != nil {
		return err
	}

	image, content, err := getItemThumbnail(s.store.db, itemId)
	if err != nil {
		return err
	}
	defer content.Close()

	return writeImage(w, image, content)
}

func (s *Server) uploadItemThumbnail(w http.ResponseWriter, r *http.Request) error {
	itemId, err := getPathId("id", r)
	if err != nil {
		return err
	}

	image, err := uploadItemThumbnail(s.store.db, itemId, r.Body)
	if err != nil {
		return err
	}

	return writeJSON(w, http.StatusOK, image)
}

func (s *Server) getImageById(w http.ResponseWriter, r *http.Request) error {
	imageId, err := getPathId("id", r)
	if err != nil {
		return err
	}

	image, content, err := getImageById(s.store.db, imageId)
	if err != nil {
		return err
	}
	defer content.Close()

	return writeImage(w, image, content)
}
