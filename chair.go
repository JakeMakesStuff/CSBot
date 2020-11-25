package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/math/fixed"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/andersfylling/disgord"
	"github.com/golang/freetype"
	"github.com/sirupsen/logrus"
	"golang.org/x/image/font"

	vision "cloud.google.com/go/vision/apiv1"
	pb "google.golang.org/genproto/googleapis/cloud/vision/v1"
)

var visionClient, visionClientErr = vision.NewImageAnnotatorClient(context.Background())

var ttfont *truetype.Font

// Initialise the propeller/mongodb client.
func init() {
	if visionClientErr != nil {
		fmt.Println("Unable to load cloud vision API:", visionClientErr.Error())
	}
	fontBytes, err := ioutil.ReadFile("impact.ttf")
	if err != nil {
		panic(err)
	}
	ttfont, err = freetype.ParseFont(fontBytes)
	if err != nil {
		panic(err)
	}
}

// RenderText is used to render the text into an image.
func RenderText(Text string, FontSize int) *image.RGBA {
	// Create the font image.
	f := float64(FontSize)
	d := &font.Drawer{
		Dst: nil,
		Src: image.White,
		Face: truetype.NewFace(ttfont, &truetype.Options{
			Size: f,
			DPI:  72,
		}),
		Dot: fixed.P(10, FontSize),
	}
	ad := d.MeasureString(Text)
	FontImg := image.NewRGBA(image.Rect(0, 0, ad.Ceil()+20, FontSize+(FontSize/2)))
	d.Dst = FontImg
	d.DrawString(Text)

	// Return the font image.
	return FontImg
}

// Handles a new message.
func messageCreateChair(s disgord.Session, evt *disgord.MessageCreate) {
	if visionClient == nil {
		// It isn't configured.
		return
	}
	if evt.Message.Author.Bot || len(evt.Message.Attachments) == 0 {
		// ignore this - is a bot or no attachments
		return
	}
	go func() {
		// Check for any images.
		images := make([]*disgord.Attachment, 0, len(evt.Message.Attachments))
		exts := []string{"png", "jpg", "jpeg"}
		for _, v := range evt.Message.Attachments {
			filenameLower := strings.ToLower(v.Filename)
			for _, ext := range exts {
				if strings.HasSuffix(filenameLower, ext) {
					images = append(images, v)
					break
				}
			}
		}
		if len(images) == 0 {
			return
		}

		// Defines the images.
		edited := make([][]byte, 0, len(images))

		// Check if we need to handle each image.
		for _, imgMetadata := range images {
			// Get the vision image reader.
			resp, err := http.Get(imgMetadata.URL)
			if err != nil {
				// Discord fucked up here.
				logrus.Error("Discord image get fail:", err)
				return
			}
			img, err := vision.NewImageFromReader(resp.Body)
			if err != nil {
				// Hmmm weird.
				logrus.Error(err)
				return
			}
			resp.Body.Close()

			// Get all animal crops in the image.
			crops, err := visionClient.LocalizeObjects(context.TODO(), img, nil)
			if err != nil {
				// Hmmm weird.
				logrus.Error(err)
				return
			}
			chairs := make([]struct {
				couch  bool
				region *pb.LocalizedObjectAnnotation
			}, 0, len(crops))
			for _, v := range crops {
				if v.Name == "Chair" || v.Name == "Couch" {
					chairs = append(chairs, struct {
						couch  bool
						region *pb.LocalizedObjectAnnotation
					}{couch: v.Name == "Couch", region: v})
				}
			}
			if len(chairs) == 0 {
				continue
			}

			// Decode the image locally.
			var imgObjUncasted image.Image
			if strings.HasSuffix(strings.ToLower(imgMetadata.URL), "png") {
				imgObjUncasted, err = png.Decode(bytes.NewReader(img.Content))
			} else {
				imgObjUncasted, err = jpeg.Decode(bytes.NewReader(img.Content))
			}
			if err != nil {
				logrus.Error(err)
				return
			}
			imgObj, ok := imgObjUncasted.(draw.Image)
			if !ok {
				imgObj = image.NewRGBA(imgObjUncasted.Bounds())
				draw.Draw(imgObj, imgObj.Bounds(), imgObjUncasted, image.Point{}, draw.Src)
			}

			// Create the rectangle for each animal.
			ImageX := imgObj.Bounds().Dx()
			ImageY := imgObj.Bounds().Dy()
			for _, chair := range chairs {
				LowestX := 9999999999
				LowestY := 9999999999
				HighestX := 0
				HighestY := 0
				for _, verts := range chair.region.BoundingPoly.NormalizedVertices {
					RealY := int(verts.Y * float32(ImageY))
					RealX := int(verts.X * float32(ImageX))
					if LowestX > RealX {
						LowestX = RealX
					}
					if LowestY > RealY {
						LowestY = RealY
					}
					if RealX > HighestX {
						HighestX = RealX
					}
					if RealY > HighestY {
						HighestY = RealY
					}
				}
				rect := image.Rect(
					LowestX,
					LowestY,
					HighestX,
					HighestY,
				)

				// Crop this part out.
				rgba := image.NewRGBA(rect)
				draw.Draw(rgba, rect, imgObj, rect.Min, draw.Over)

				// Create the font image.
				label := "CHAIR"
				if chair.couch {
					label = "TORY " + label
				}
				rendered := RenderText(label, rect.Dy()/10)

				// Create the point.
				pt := image.Point{}
				pt.X -= rgba.Bounds().Dx() / 2
				pt.X += rendered.Bounds().Dx() / 2

				// Draw the text.
				draw.Draw(rgba, rect, rendered, pt, draw.Over)

				// Encode as a PNG.
				buf := &bytes.Buffer{}
				err = png.Encode(buf, rgba)
				if err != nil {
					panic(err)
				}
				edited = append(edited, buf.Bytes())
			}
		}

		// If the length of edited isn't 0, send the chairs.
		if len(edited) != 0 {
			data := make([]interface{}, len(edited))
			for i, img := range edited {
				data[i] = &disgord.CreateMessageFileParams{
					Reader:     bytes.NewReader(img),
					FileName:   "chairs.png",
					SpoilerTag: false,
				}
			}
			_, _ = s.SendMsg(context.TODO(), evt.Message.ChannelID, data...)
		}
	}()
}
