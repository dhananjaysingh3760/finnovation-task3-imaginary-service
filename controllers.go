package main

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/h2non/bimg"
	"github.com/h2non/filetype"
)

type ReqBody struct {
	UserId                    string `json:"userId"`
	BucketName                string `json:"bucketName"`
	FolderPath                string `json:"folderPath"`
	ClientResourceStorageName string `json:"clientResourceStorageName"`
}

func indexController(o ServerOptions) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != path.Join(o.PathPrefix, "/") {
			s := r.URL.Path
			i := 0
			if s[1:] == "register" {
				body := ReqBody{}
				json.NewDecoder(r.Body).Decode(&body)
				fmt.Println(body)
				userId = body.UserId
				register(body.ClientResourceStorageName, body.BucketName, body.FolderPath, w, r)
				return
			}
			userId = ""
			for i = 1; ; i++ {
				if s[i] == '/' {
					break
				}
				yo := string([]rune(s)[i])
				userId = userId + yo
			}
			fmt.Println(userId)
			filePath = ""
			for ; i < len(s); i++ {
				if s[i] == '\n' {
					break
				}
				yo := string([]rune(s)[i])
				filePath = filePath + yo
			}
			q := r.URL.Query()
			q.Add("url", "just chill kk")
			r.URL.RawQuery = q.Encode()
			var imageSource = MatchSource(r)
			if imageSource == nil {
				ErrorReply(r, w, ErrMissingImageSource, o)
				return
			}
			buf, err := imageSource.GetImage(r)
			if err != nil {
				if xerr, ok := err.(Error); ok {
					ErrorReply(r, w, xerr, o)
				} else {
					ErrorReply(r, w, NewError(err.Error(), http.StatusBadRequest), o)
				}
				return
			}

			if len(buf) == 0 {
				ErrorReply(r, w, ErrEmptyBody, o)
				return
			}
			s2 := r.URL.Query()
			s1 := s2["method"][0]
			switch s1 {
			case "thumbnail":
				imageHandler(w, r, buf, Thumbnail, o)
			case "resize":
				imageHandler(w, r, buf, Resize, o)
			case "fit":
				imageHandler(w, r, buf, Fit, o)
			case "enlarge":
				imageHandler(w, r, buf, Enlarge, o)
			case "extract":
				imageHandler(w, r, buf, Extract, o)
			case "crop":
				imageHandler(w, r, buf, Crop, o)
			case "smartcrop":
				imageHandler(w, r, buf, SmartCrop, o)
			case "rotate":
				imageHandler(w, r, buf, Rotate, o)
			case "autorotate":
				imageHandler(w, r, buf, AutoRotate, o)
			case "flip":
				imageHandler(w, r, buf, Flip, o)
			case "flop":
				imageHandler(w, r, buf, Flop, o)
			case "zoom":
				imageHandler(w, r, buf, Zoom, o)
			case "convert":
				imageHandler(w, r, buf, Convert, o)
			case "watermark":
				imageHandler(w, r, buf, Watermark, o)
			case "watermarkimage":
				imageHandler(w, r, buf, WatermarkImage, o)
			case "info":
				imageHandler(w, r, buf, Info, o)
			case "blur":
				imageHandler(w, r, buf, GaussianBlur, o)
			case "pipeline":
				imageHandler(w, r, buf, Pipeline, o)
			}
		}

		body, _ := json.Marshal(Versions{
			Version,
			bimg.Version,
			bimg.VipsVersion,
		})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}
}

func healthController(w http.ResponseWriter, r *http.Request) {
	health := GetHealthStats()
	body, _ := json.Marshal(health)
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}

func imageController(o ServerOptions, operation Operation) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		var imageSource = MatchSource(req)
		if imageSource == nil {
			ErrorReply(req, w, ErrMissingImageSource, o)
			return
		}

		buf, err := imageSource.GetImage(req)
		if err != nil {
			if xerr, ok := err.(Error); ok {
				ErrorReply(req, w, xerr, o)
			} else {
				ErrorReply(req, w, NewError(err.Error(), http.StatusBadRequest), o)
			}
			return
		}

		if len(buf) == 0 {
			ErrorReply(req, w, ErrEmptyBody, o)
			return
		}

		imageHandler(w, req, buf, operation, o)
	}
}

func determineAcceptMimeType(accept string) string {
	for _, v := range strings.Split(accept, ",") {
		mediaType, _, _ := mime.ParseMediaType(v)
		switch mediaType {
		case "image/webp":
			return "webp"
		case "image/png":
			return "png"
		case "image/jpeg":
			return "jpeg"
		}
	}

	return ""
}

func imageHandler(w http.ResponseWriter, r *http.Request, buf []byte, operation Operation, o ServerOptions) {
	// Infer the body MIME type via mime sniff algorithm
	mimeType := http.DetectContentType(buf)

	// If cannot infer the type, infer it via magic numbers
	if mimeType == "application/octet-stream" {
		kind, err := filetype.Get(buf)
		if err == nil && kind.MIME.Value != "" {
			mimeType = kind.MIME.Value
		}
	}

	// Infer text/plain responses as potential SVG image
	if strings.Contains(mimeType, "text/plain") && len(buf) > 8 {
		if bimg.IsSVGImage(buf) {
			mimeType = "image/svg+xml"
		}
	}

	// Finally check if image MIME type is supported
	if !IsImageMimeTypeSupported(mimeType) {
		ErrorReply(r, w, ErrUnsupportedMedia, o)
		return
	}

	opts, err := buildParamsFromQuery(r.URL.Query())
	if err != nil {
		ErrorReply(r, w, NewError("Error while processing parameters, "+err.Error(), http.StatusBadRequest), o)
		return
	}

	vary := ""
	if opts.Type == "auto" {
		opts.Type = determineAcceptMimeType(r.Header.Get("Accept"))
		vary = "Accept" // Ensure caches behave correctly for negotiated content
	} else if opts.Type != "" && ImageType(opts.Type) == 0 {
		ErrorReply(r, w, ErrOutputFormat, o)
		return
	}

	image, err := operation.Run(buf, opts)
	if err != nil {
		// Ensure the Vary header is set when an error occurs
		if vary != "" {
			w.Header().Set("Vary", vary)
		}
		ErrorReply(r, w, NewError("Error while processing the image: "+err.Error(), http.StatusBadRequest), o)
		return
	}

	// Expose Content-Length response header
	w.Header().Set("Content-Length", strconv.Itoa(len(image.Body)))
	w.Header().Set("Content-Type", image.Mime)
	if vary != "" {
		w.Header().Set("Vary", vary)
	}
	_, _ = w.Write(image.Body)
}

func formController(o ServerOptions) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		operations := []struct {
			name   string
			method string
			args   string
		}{
			{"Resize", "resize", "width=300&height=200&type=jpeg"},
			{"Force resize", "resize", "width=300&height=200&force=true"},
			{"Crop", "crop", "width=300&quality=95"},
			{"SmartCrop", "crop", "width=300&height=260&quality=95&gravity=smart"},
			{"Extract", "extract", "top=100&left=100&areawidth=300&areaheight=150"},
			{"Enlarge", "enlarge", "width=1440&height=900&quality=95"},
			{"Rotate", "rotate", "rotate=180"},
			{"AutoRotate", "autorotate", "quality=90"},
			{"Flip", "flip", ""},
			{"Flop", "flop", ""},
			{"Thumbnail", "thumbnail", "width=100"},
			{"Zoom", "zoom", "factor=2&areawidth=300&top=80&left=80"},
			{"Color space (black&white)", "resize", "width=400&height=300&colorspace=bw"},
			{"Add watermark", "watermark", "textwidth=100&text=Hello&font=sans%2012&opacity=0.5&color=255,200,50"},
			{"Convert format", "convert", "type=png"},
			{"Image metadata", "info", ""},
			{"Gaussian blur", "blur", "sigma=15.0&minampl=0.2"},
			{"Pipeline (image reduction via multiple transformations)", "pipeline", "operations=%5B%7B%22operation%22:%20%22crop%22,%20%22params%22:%20%7B%22width%22:%20300,%20%22height%22:%20260%7D%7D,%20%7B%22operation%22:%20%22convert%22,%20%22params%22:%20%7B%22type%22:%20%22webp%22%7D%7D%5D"},
		}

		html := "<html><body>"

		for _, form := range operations {
			html += fmt.Sprintf(`
		<h1>%s</h1>
		<form method="POST" action="%s?%s" enctype="multipart/form-data">
		<input type="file" name="file" />
		<input type="submit" value="Upload" />
		</form>`, path.Join(o.PathPrefix, form.name), path.Join(o.PathPrefix, form.method), form.args)
		}

		html += "</body></html>"

		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(html))
	}
}
