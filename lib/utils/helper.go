package utils

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/smtp"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"

	"golang.org/x/net/html"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/globalsign/mgo/bson"
	"github.com/spf13/viper"
)

type Request struct {
	from    string
	to      []string
	subject string
	body    string
}

type (
	SliceNumber struct {
		Uint     []uint
		Uint32   []uint32
		Uint64   []uint64
		Uint16   []uint16
		Int      []int
		Int16    []int16
		Int32    []int32
		Int64    []int64
		IsSorted bool
		SliceRange
		SMin   int64
		SMax   int64
		ORange [][]int64
	}

	SliceRange [][]int64
)

const (
	MIME = "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
)

func Body(doc *html.Node) (*html.Node, error) {
	var body *html.Node
	var crawler func(*html.Node)
	crawler = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "body" {
			body = node
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			crawler(child)
		}
	}
	crawler(doc)
	if body != nil {
		return body, nil
	}
	return nil, errors.New("Missing <body> in the node tree")
}

func renderNode(n *html.Node) string {
	var buf bytes.Buffer
	w := io.Writer(&buf)
	html.Render(w, n)
	return buf.String()
}

func NewRequestSendMail(to []string, subject string) *Request {
	return &Request{
		to:      to,
		subject: subject,
	}
}

func (r *Request) parseTemplate(htmlText string) error {
	// t, err := template.ParseFiles(fileName)
	// if err != nil {
	// 	return err
	// }
	// buffer := new(bytes.Buffer)
	// if err = t.Execute(buffer, data); err != nil {
	// 	return err
	// }
	doc, _ := html.Parse(strings.NewReader(htmlText))
	bn, err := Body(doc)
	if err != nil {
		fmt.Println(err)
	}
	body := renderNode(bn)
	// r.body = buffer.String()
	r.body = body

	return nil
}

func (r *Request) Send(htmlText string) {
	// err, _ := html.Parse(strings.NewReader(templateName))
	err := r.parseTemplate(htmlText)
	if err != nil {
		log.Fatal(err)
	}
	if ok := r.sendMail(); ok {
		log.Printf("Email has been sent to %s\n", r.to)
	} else {
		log.Printf("Failed to send the email to %s\n", r.to)
	}
}

func (r *Request) sendMail() bool {
	for _, to := range r.to {
		body := "To: " + to + "\r\nFrom: " + viper.GetString("mail.MAIL_FROM_ADDRESS") + "\r\nSubject: " + r.subject + "\r\n" + MIME + "\r\n" + r.body
		SMTP := fmt.Sprintf("%s:%d", viper.GetString("mail.MAIL_HOST"), 587)

		email := []string{to}
		if err := smtp.SendMail(SMTP, smtp.PlainAuth("", viper.GetString("mail.MAIL_USERNAME"), viper.GetString("mail.MAIL_PASSWORD"), viper.GetString("mail.MAIL_HOST")), viper.GetString("mail.MAIL_FROM_ADDRESS"), email, []byte(body)); err != nil {
			fmt.Println(err)
			return false
		}
	}

	return true

}

// UploadFileToS3 saves a file to aws bucket and returns the url to // the file and an error if there's any
func UploadFileToS3(s *session.Session, typeImage string, file multipart.File, fileHeader *multipart.FileHeader) (string, error) {
	// get the file size and read
	// the file content into a buffer
	size := fileHeader.Size
	buffer := make([]byte, size)
	file.Read(buffer)

	// create a unique file name for the file
	tempFileName := typeImage + "/" + bson.NewObjectId().Hex() + filepath.Ext(fileHeader.Filename)

	// config settings: this is where you choose the bucket,
	// filename, content-type and storage class of the file
	// you're uploading
	_, err := s3.New(s).PutObject(&s3.PutObjectInput{
		Bucket:               aws.String(viper.GetString("aws.S3_BUCKET")),
		Key:                  aws.String(tempFileName),
		ACL:                  aws.String("public-read"), // could be private if you want it to be access by only authorized users
		Body:                 bytes.NewReader(buffer),
		ContentLength:        aws.Int64(int64(size)),
		ContentType:          aws.String(http.DetectContentType(buffer)),
		ContentDisposition:   aws.String("attachment"),
		ServerSideEncryption: aws.String("AES256"),
		StorageClass:         aws.String("INTELLIGENT_TIERING"),
	})
	if err != nil {
		return "", err
	}

	return tempFileName, err
}

func UploadFile(w http.ResponseWriter, r *http.Request, typeImage string) string {

	maxSize := int64(1024000) // allow only 1MB of file size
	err := r.ParseMultipartForm(maxSize)

	if err != nil {
		log.Println(err)
		fmt.Fprintf(w, "Image too large. Max Size: %v", maxSize)
		return ""
	}
	file, fileHeader, err := r.FormFile("file")

	if err != nil {
		log.Println(err)
		fmt.Fprintf(w, "Could not get uploaded file")
		return ""
	}
	defer file.Close()
	// create an AWS session which can be
	// reused if we're uploading many files
	s, err := session.NewSession(&aws.Config{
		Region: aws.String(viper.GetString("aws.S3_REGION")),
		Credentials: credentials.NewStaticCredentials(
			viper.GetString("aws.aws_key"),    // id
			viper.GetString("aws.aws_secret"), // secret
			""),                               // token can be left blank for now
	})
	if err != nil {
		fmt.Fprintf(w, "Could not upload file")
	}
	fileName, err := UploadFileToS3(s, typeImage, file, fileHeader)
	if err != nil {
		fmt.Fprintf(w, "Could not upload file")
	}
	// fmt.Fprintf(w, "Image uploaded successfully: %v", fileName)

	return fileName

}

func CompareMemberCode(mCodeJwt string, mCodeReq string) bool {

	if mCodeJwt != mCodeReq {
		return false
	}

	return true
}

func PrintMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	log.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	log.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	log.Printf("\tSys = %v MiB", bToMb(m.Sys))
	log.Printf("\tNumGC = %v\n", m.NumGC)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

/**
 * Range function for range of slice on SliceRange with decide number list
 * @Parameter:
 *      slice<T>.(reflect.Slice)
 *      for execute the sorted slice into range of uint or int
 */
func (rg *SliceRange) Range(slice interface{}) error {
	rf := reflect.ValueOf(slice)
	if rf.Kind() != reflect.Slice {
		return fmt.Errorf("not a slice, please insert a slice type")
	}

	ts := make([]int64, 0)
	var trg [][]int64
	for i := 0; i < rf.Len(); i++ {
		v := rf.Index(i).Interface()
		val := reflect.ValueOf(v)

		inx := int64(0)
		switch val.Kind() {
		case reflect.Uint:
			inx = int64(v.(uint))
		case reflect.Uint16:
			inx = int64(v.(uint16))
		case reflect.Uint32:
			inx = int64(v.(uint32))
		case reflect.Uint64:
			inx = int64(v.(uint64))
		case reflect.Int:
			inx = int64(v.(int))
		case reflect.Int16:
			inx = int64(v.(int16))
		case reflect.Int32:
			inx = int64(v.(int32))
		case reflect.Int64:
			inx = v.(int64)
		}

		if i < 1 {
			trg = append(trg, []int64{inx})
		} else {
			inz := ts[len(ts)-1]
			inc := inx - inz

			switch {
			case inc < 1:
				continue
			case inc == 1:
				if len(trg) > 0 {
					lasts := trg[len(trg)-1]
					if len(lasts) < 2 {
						lasts = append(lasts, inx)
					} else {
						lasts[len(lasts)-1] = inx
					}
					trg[len(trg)-1] = lasts
				}
			case inc > 1:
				trg = append(trg, []int64{inx})
			}
		}

		ts = append(ts, inx)
	}

	*rg = trg
	return nil
}

/**
 * Sort function for create ArrayInt into sorted value
 * @Parameter:
 *      have one input [Duplicate<boolean>] for flag
 *      if true will return sorted SliceUint.<T> with duplicate value (if have any duplicate)
 *      if false will return sorted SliceUint.<T> without duplicate value
 */
func (su *SliceNumber) Sort(duplicate bool) {
	su.IsSorted = true
	switch {
	case len(su.Uint) > 0:
		if !sort.SliceIsSorted(su.Uint, func(i, j int) bool { return su.Uint[i] < su.Uint[j] }) {
			sort.Slice(su.Uint, func(i, j int) bool { return su.Uint[i] < su.Uint[j] })
		}

		if !duplicate {
			var u []uint
			for i := range su.Uint {
				if i < 1 {
					u = append(u, su.Uint[i])
				} else {
					if len(u) > 0 && su.Uint[i] != u[len(u)-1] {
						u = append(u, su.Uint[i])
					}
				}
			}
			su.Uint = u
		}
		su.SMin = int64(su.Uint[0])
		su.SMax = int64(su.Uint[len(su.Uint)-1])
	case len(su.Uint16) > 0:
		if !sort.SliceIsSorted(su.Uint16, func(i, j int) bool { return su.Uint16[i] < su.Uint16[j] }) {
			sort.Slice(su.Uint16, func(i, j int) bool { return su.Uint16[i] < su.Uint16[j] })
		}

		if !duplicate {
			var u []uint16
			for i := range su.Uint16 {
				if i < 1 {
					u = append(u, su.Uint16[i])
				} else {
					if len(u) > 0 && su.Uint16[i] != u[len(u)-1] {
						u = append(u, su.Uint16[i])
					}
				}
			}
			su.Uint16 = u
		}
		su.SMin = int64(su.Uint16[0])
		su.SMax = int64(su.Uint16[len(su.Uint16)-1])
	case len(su.Uint32) > 0:
		if !sort.SliceIsSorted(su.Uint32, func(i, j int) bool { return su.Uint32[i] < su.Uint32[j] }) {
			sort.Slice(su.Uint32, func(i, j int) bool { return su.Uint32[i] < su.Uint32[j] })
		}

		if !duplicate {
			var u []uint32
			for i := range su.Uint32 {
				if i < 1 {
					u = append(u, su.Uint32[i])
				} else {
					if len(u) > 0 && su.Uint32[i] != u[len(u)-1] {
						u = append(u, su.Uint32[i])
					}
				}
			}
			su.Uint32 = u
		}
		su.SMin = int64(su.Uint32[0])
		su.SMax = int64(su.Uint32[len(su.Uint32)-1])
	case len(su.Uint64) > 0:
		if !sort.SliceIsSorted(su.Uint64, func(i, j int) bool { return su.Uint64[i] < su.Uint64[j] }) {
			sort.Slice(su.Uint64, func(i, j int) bool { return su.Uint64[i] < su.Uint64[j] })
		}

		if !duplicate {
			var u []uint64
			for i := range su.Uint64 {
				if i < 1 {
					u = append(u, su.Uint64[i])
				} else {
					if len(u) > 0 && su.Uint64[i] != u[len(u)-1] {
						u = append(u, su.Uint64[i])
					}
				}
			}
			su.Uint64 = u
		}
		su.SMin = int64(su.Uint64[0])
		su.SMax = int64(su.Uint64[len(su.Uint64)-1])
	case len(su.Int) > 0:
		if !sort.SliceIsSorted(su.Int, func(i, j int) bool { return su.Int[i] < su.Int[j] }) {
			sort.Slice(su.Int, func(i, j int) bool { return su.Int[i] < su.Int[j] })
		}

		if !duplicate {
			var u []int
			for i := range su.Int {
				if i < 1 {
					u = append(u, su.Int[i])
				} else {
					if len(u) > 0 && su.Int[i] != u[len(u)-1] {
						u = append(u, su.Int[i])
					}
				}
			}
			su.Int = u
		}
		su.SMin = int64(su.Int[0])
		su.SMax = int64(su.Int[len(su.Int)-1])
	case len(su.Int16) > 0:
		if !sort.SliceIsSorted(su.Int16, func(i, j int) bool { return su.Int16[i] < su.Int16[j] }) {
			sort.Slice(su.Int16, func(i, j int) bool { return su.Int16[i] < su.Int16[j] })
		}

		if !duplicate {
			var u []int16
			for i := range su.Int16 {
				if i < 1 {
					u = append(u, su.Int16[i])
				} else {
					if len(u) > 0 && su.Int16[i] != u[len(u)-1] {
						u = append(u, su.Int16[i])
					}
				}
			}
			su.Int16 = u
		}
		su.SMin = int64(su.Int16[0])
		su.SMax = int64(su.Int16[len(su.Int16)-1])
	case len(su.Int32) > 0:
		if !sort.SliceIsSorted(su.Int32, func(i, j int) bool { return su.Int32[i] < su.Int32[j] }) {
			sort.Slice(su.Int32, func(i, j int) bool { return su.Int32[i] < su.Int32[j] })
		}

		if !duplicate {
			var u []int32
			for i := range su.Int32 {
				if i < 1 {
					u = append(u, su.Int32[i])
				} else {
					if len(u) > 0 && su.Int32[i] != u[len(u)-1] {
						u = append(u, su.Int32[i])
					}
				}
			}
			su.Int32 = u
		}
		su.SMin = int64(su.Int32[0])
		su.SMax = int64(su.Int32[len(su.Int32)-1])
	case len(su.Int64) > 0:
		if !sort.SliceIsSorted(su.Int64, func(i, j int) bool { return su.Int64[i] < su.Int64[j] }) {
			sort.Slice(su.Int64, func(i, j int) bool { return su.Int64[i] < su.Int64[j] })
		}

		if !duplicate {
			var u []int64
			for i := range su.Int64 {
				if i < 1 {
					u = append(u, su.Int64[i])
				} else {
					if len(u) > 0 && su.Int64[i] != u[len(u)-1] {
						u = append(u, su.Int64[i])
					}
				}
			}
			su.Int64 = u
		}
		su.SMin = su.Int64[0]
		su.SMax = su.Int64[len(su.Int64)-1]
	}
}

/**
 * Append function for append value into ArrayInt data type
 * This saved slice what your want to append into property of SliceUInt
 */
func (su *SliceNumber) Append(slice interface{}) error {
	sc := reflect.ValueOf(slice)
	if sc.Kind() != reflect.Slice {
		return fmt.Errorf("not a slice, please insert a slice type of uint")
	}

	for i := 0; i < sc.Len(); i++ {
		v := sc.Index(i).Interface()
		val := reflect.ValueOf(v)
		switch val.Kind() {
		case reflect.Uint16:
			su.Uint16 = append(su.Uint16, v.(uint16))
		case reflect.Uint32:
			su.Uint32 = append(su.Uint32, v.(uint32))
		case reflect.Uint:
			su.Uint = append(su.Uint, v.(uint))
		case reflect.Uint64:
			su.Uint64 = append(su.Uint64, v.(uint64))
		case reflect.Int:
			su.Int = append(su.Int, v.(int))
		case reflect.Int16:
			su.Int16 = append(su.Int16, v.(int16))
		case reflect.Int32:
			su.Int32 = append(su.Int32, v.(int32))
		case reflect.Int64:
			su.Int64 = append(su.Int64, v.(int64))
		default:
			return fmt.Errorf("not uint data type, just Uint ot Int")
		}
	}

	return nil
}

/**
 * InRange for grouping range on lists of number it will be sorting first if lists is no sorted
 */
func (su *SliceNumber) InRange() error {
	var si interface{}

	switch {
	case len(su.Uint) > 0:
		si = su.Uint
	case len(su.Uint16) > 0:
		si = su.Uint16
	case len(su.Uint32) > 0:
		si = su.Uint32
	case len(su.Uint64) > 0:
		si = su.Uint64
	case len(su.Int) > 0:
		si = su.Int
	case len(su.Int16) > 0:
		si = su.Int16
	case len(su.Int32) > 0:
		si = su.Int32
	case len(su.Int64) > 0:
		si = su.Int64
	}

	if !su.IsSorted {
		su.Sort(false)
		return su.InRange()
	}

	err := su.SliceRange.Range(si)
	if err != nil {
		return err
	}

	return nil
}

/**
 * OutRange give you for slice of result out of range on listed slice number
 * It will be generate first for Range on SliceRange
 */
func (su *SliceNumber) OutRange() ([][]int64, error) {
	var (
		fx  [][]int64
		err error
	)

	if len(su.SliceRange) < 1 {
		err = su.InRange()
		if err != nil {
			return fx, err
		}
	}

	if len(su.SliceRange) > 0 {
		// lnsr: list number slice range
		lnsr := len(su.SliceRange)
		for i := range su.SliceRange {

			// fr: first range
			fr := su.SliceRange[i]
			if i < 1 {
				if lnsr > 1 {
					fx = append(fx, []int64{fr[len(fr)-1] + 1})
				}
			} else {
				if len(fr) > 0 {
					fr1 := fr[0]
					fr2 := fr[len(fr)-1]

					if len(fx) > 0 {
						// fxl: first range x last
						fxl := fx[len(fx)-1]
						if len(fxl) > 0 {
							if fxl[0] != (fr1 - 1) {
								fxl = append(fxl, fr1-1)
								fx[len(fx)-1] = fxl
							}

							if (i + 1) < lnsr {
								fx = append(fx, []int64{fr2 + 1})
							}
						}
					}
				}
			}
		}
	}

	return fx, nil
}

// StringContainsArray check if word is exists on array
func StringContainsArray(arr []string, word string) bool {
	if len(word) > 0 {
		for _, w := range arr {
			exs := strings.Contains(w, word)
			if exs {
				return true
			}
		}
	}
	return false
}

// CheckValueExists check if one of many value is exists
func CheckValueExists(entries ...interface{}) bool {

	for _, ent := range entries {
		if ent != nil {
			rent := reflect.ValueOf(ent)
			switch rent.Kind() {
			case reflect.Array, reflect.Map:
				if rent.Len() > 0 {
					return true
				}
			default:
				if !rent.IsZero() {
					return true
				}
			}
		}
	}

	return false
}
