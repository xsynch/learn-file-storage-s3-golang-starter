package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	// fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()
	mediaType := header.Header.Get("Content-Type")
	if mediaType == "" {
		respondWithError(w, http.StatusBadRequest, "Content type is empty", nil)
		return
	}
	// fmt.Println("the media type is ",mediaType)

	fileData, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "unable to get media info", err)
		return
	}
	vidData, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "unable to get video data", err)
		return
	}
	if vidData.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "User not authorized to update this media data", nil)
		return
	}

	encodedVideo := base64.StdEncoding.EncodeToString(fileData)
	dataUrl := fmt.Sprintf("data:%s;base64,%s", mediaType, encodedVideo)

	// url := fmt.Sprintf("localhost:%s/api/thumbnails/%v", cfg.port, videoID)
	vidData.ThumbnailURL = &dataUrl

	// newThumbnail := thumbnail{data: fileData, mediaType: mediaType}
	// videoThumbnails[videoID] = newThumbnail
	err = cfg.db.UpdateVideo(vidData)
	if err != nil {
		// delete(videoThumbnails, videoID)
		respondWithError(w, http.StatusBadRequest, "unable to update video information", err)
		return
	}

	respondWithJSON(w, http.StatusOK, vidData)
}
