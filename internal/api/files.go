package api

import (
	"net/http"
	"path/filepath"

	"github.com/SergeyKozhin/shared-planner-backend/internal/config"
)

func (a *Api) uploadFileHandler(w http.ResponseWriter, r *http.Request) {
	multipartFile, headers, err := r.FormFile("file")
	if err != nil {
		a.badRequestResponse(w, r, err)
		return
	}
	defer multipartFile.Close()

	if headers.Size > config.MaxFileSize() {
		a.fileTooBigResponse(w, r)
		return
	}

	filePath, err := savePicture(multipartFile, filepath.Ext(headers.Filename))
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	resp := &attachment{
		Name: headers.Filename,
		Path: filePath,
	}
	if err := a.writeJSON(w, http.StatusCreated, resp, nil); err != nil {
		a.serverErrorResponse(w, r, err)
	}
}
