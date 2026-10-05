package handlers

import (
	"bytes"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/asihat/b2b-marketplace/backend/internal/httpx"
	"github.com/asihat/b2b-marketplace/backend/internal/strx"
	"github.com/asihat/b2b-marketplace/backend/internal/validate"
)

func (h *Handlers) AdminSettings(w http.ResponseWriter, r *http.Request) error {
	httpx.JSON(w, http.StatusOK, h.app.Settings.Public(r.Context()))
	return nil
}

func (h *Handlers) AdminSettingsUpdate(w http.ResponseWriter, r *http.Request) error {
	data, err := httpx.Input(r, false)
	if err != nil {
		return err
	}
	v := validate.New(data)
	values := map[string]*string{}

	if v.Has("mode") {
		if s, ok := v.String("mode", validate.Str{Required: true, In: []string{"b2c", "b2b"}}); ok {
			values["mode"] = &s
		}
	}
	if v.Has("company_name") {
		if s, ok := v.String("company_name", validate.Str{Required: true, Max: 120}); ok {
			values["company_name"] = &s
		}
	}
	if v.Has("company_description") {
		if s, present := v.OptString("company_description", validate.Str{Max: 500}); present {
			values["company_description"] = s
		}
	}
	if err := v.Err(); err != nil {
		return err
	}

	settings, err := h.app.Settings.Update(r.Context(), values)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, settings)
	return nil
}

const maxIconBytes = 2048 * 1024

var iconTypes = map[string]string{
	"image/png":  "png",
	"image/jpeg": "jpg",
	"image/webp": "webp",
}

func (h *Handlers) AdminSettingsUploadIcon(w http.ResponseWriter, r *http.Request) error {
	v := validate.New(map[string]any{})
	if err := r.ParseMultipartForm(maxIconBytes + 1024); err != nil {
		v.Fail("icon", "The icon field is required.")
		return v.Err()
	}
	file, header, err := r.FormFile("icon")
	if err != nil {
		v.Fail("icon", "The icon field is required.")
		return v.Err()
	}
	defer file.Close()

	content, err := io.ReadAll(io.LimitReader(file, maxIconBytes+1))
	if err != nil {
		return err
	}
	if len(content) > maxIconBytes {
		v.Fail("icon", "The icon field must not be greater than 2048 kilobytes.")
		return v.Err()
	}
	detected := http.DetectContentType(content)
	ext, ok := iconTypes[detected]
	extOK := map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".webp": true}[strings.ToLower(path.Ext(header.Filename))]
	if !ok {
		v.Fail("icon", "The icon field must be an image.")
		return v.Err()
	}
	if !extOK {
		v.Fail("icon", "The icon field must be a file of type: png, jpg, jpeg, webp.")
		return v.Err()
	}

	rel := "settings/icons/" + strx.Random(40) + "." + ext
	if err := h.app.Disk.Put(rel, bytes.NewReader(content)); err != nil {
		return httpx.NewError(http.StatusInternalServerError, "Icon could not be stored.")
	}

	settings, err := h.app.Settings.UpdateIcon(r.Context(), rel)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, settings)
	return nil
}

func (h *Handlers) AdminSettingsRemoveIcon(w http.ResponseWriter, r *http.Request) error {
	settings, err := h.app.Settings.RemoveIcon(r.Context())
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, settings)
	return nil
}
