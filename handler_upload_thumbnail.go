package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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
	// mediaType := header.Header.Get("Content-Type")
	// if mediaType == "" {
	// 	respondWithError(w, http.StatusBadRequest, "Content type is empty", nil)
	// 	return
	// }
	mediaType,_,err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w,http.StatusBadRequest,"Error Parsing Media Type",err)
		return 
	}
	if mediaType == "" {
		respondWithError(w, http.StatusBadRequest, "Content type is empty", nil)
		return
	}
	if (mediaType != "image/jpeg" && mediaType != "image/png"){
		respondWithError(w,http.StatusBadRequest,"Only Images Allowed",nil)
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
	

	fileType := strings.Split(mediaType,"/")
	filePath := filepath.Join(cfg.assetsRoot,fmt.Sprintf("%v.%s",videoID,fileType[1]))
	outfile, err := os.Create(filePath)
	if err != nil {
		respondWithError(w,http.StatusBadRequest,"Error creating the file",err)
		return 
	}
	defer outfile.Close()
	
	
	_, err = io.Copy(outfile,file)
	if err != nil {
		respondWithError(w,http.StatusBadRequest,"Error copying data to the file",err)
		return 
	}
	fileUrl := fmt.Sprintf("http://localhost:%s/%s",cfg.port,filePath)

	vidData.ThumbnailURL = &fileUrl

	err = cfg.db.UpdateVideo(vidData)
	if err != nil {
		
		respondWithError(w, http.StatusBadRequest, "unable to update video information", err)
		return
	}

	respondWithJSON(w, http.StatusOK, vidData)
}
