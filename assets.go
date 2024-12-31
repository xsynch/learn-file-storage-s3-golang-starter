package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	
)


type VideoMetadata struct {
	Streams []struct {
		Index              int    `json:"index"`
		CodecName          string `json:"codec_name,omitempty"`
		CodecLongName      string `json:"codec_long_name,omitempty"`
		Profile            string `json:"profile,omitempty"`
		CodecType          string `json:"codec_type"`
		CodecTagString     string `json:"codec_tag_string"`
		CodecTag           string `json:"codec_tag"`
		Width              int    `json:"width,omitempty"`
		Height             int    `json:"height,omitempty"`
		CodedWidth         int    `json:"coded_width,omitempty"`
		CodedHeight        int    `json:"coded_height,omitempty"`	
		DisplayAspectRatio string `json:"display_aspet_ratio"`	
	} `json:"streams"`
}






func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0755)
	}
	return nil
}


func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command("ffprobe","-v", "error","-print_format", "json","-show_streams",filePath)
	out := bytes.Buffer{}
	cmd.Stdout = &out 
	err := cmd.Run()
	if err != nil {
		return "",err 
	}
	videoMetadata := VideoMetadata{}

	err = json.Unmarshal(out.Bytes(),&videoMetadata)
	if err != nil {
		return "",err 
	}


	
	for _,val := range videoMetadata.Streams{
		
		

		
		if val.Width > val.Height {
			aspectRatio := fmt.Sprintf("%.2f",float32(val.Width)/float32(val.Height))
			// fmt.Printf("Aspect Ratio Larger Width: %s\n",aspectRatio)
			if aspectRatio == "1.78"{
				return "16:9",nil 
			} else {
				return "other",nil 
			}
		}

		if val.Width < val.Height {
			aspectRatio := fmt.Sprintf("%.2f",float32(val.Height)/float32(val.Width))
			// fmt.Printf("Aspect Ratio Larger Height: %s\n",aspectRatio)
			if aspectRatio == "1.78"{
				return "9:16",nil 
			} else {
				return "other",nil 
			}
		}

	}

	return "",fmt.Errorf("unable to get aspect ratio")



}

func processVideoForFastStart(filePath string) (string, error) {
	newPath := fmt.Sprintf("%s.processing",filePath)
	cmd := exec.Command("ffmpeg","-i",filePath,"-c","copy","-movflags","faststart","-f","mp4",newPath)
	err := cmd.Run()
	if err != nil {
		return "",err 
	}

	return newPath,nil 
}