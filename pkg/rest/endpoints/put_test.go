package endpoints

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"net/http"

	"bytes"
	"encoding/json"
	"github.com/emicklei/go-restful"
	"golang.org/x/net/context"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"kubevirt.io/kubevirt/pkg/rest"
	"net/http/httptest"
	"net/url"
	"strings"
)

func marshalToJSON(payload interface{}) io.ReadCloser {
	var b []byte
	buffer := bytes.NewBuffer(b)
	Expect(json.NewEncoder(buffer).Encode(payload)).To(Succeed())
	return ioutil.NopCloser(buffer)
}

func marshalToYAML(payload interface{}) io.ReadCloser {
	var b []byte
	buffer := bytes.NewBuffer(b)
	raw, err := yaml.Marshal(payload)
	Expect(err).ToNot(HaveOccurred())
	buffer.Write(raw)
	return ioutil.NopCloser(buffer)
}

type payload struct {
	Name     string   `json:"name" valid:"required"`
	Email    string   `json:"email" valid:"email"`
	Metadata Metadata `json:"metadata"`
}

func newValidPutRequest() *http.Request {
	request, _ := http.NewRequest("PUT", "/apis/kubevirt.io/v1alpha1/namespaces/default/vms/test", nil)
	request.Body = marshalToJSON(payload{Name: "test", Email: "test@test.com"})
	request.Header.Set("Content-Type", rest.MIME_JSON)
	return request
}

func testPutEndpoint(_ context.Context, request interface{}) (interface{}, error) {
	Expect(request.(*PutObject).Metadata.Name).To(Equal("test"))
	return request.(*PutObject).Payload, nil
}

var _ = Describe("Put", func() {
	var recorder *httptest.ResponseRecorder
	var request *http.Request
	var handler http.Handler
	ctx := context.Background()

	BeforeEach(func() {

		ws := new(restful.WebService)
		ws.Produces(restful.MIME_JSON, rest.MIME_YAML).Consumes(restful.MIME_JSON, rest.MIME_YAML)
		handler = http.Handler(restful.NewContainer().Add(ws))

		target := MakeGoRestfulWrapper(NewHandlerBuilder().Put((*payload)(nil)).Endpoint(testPutEndpoint).Build(ctx))
		ws.Route(ws.PUT("/apis/kubevirt.io/v1alpha1/namespaces/{namespace}/vms/{name}").To(target))

		request = newValidPutRequest()
		recorder = httptest.NewRecorder()
	})

	Describe("REST call", func() {
		Context("with invalid URL", func() {
			It("should return 404", func() {
				request.URL, _ = url.Parse("/api/rest/put/?")
				handler.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(http.StatusNotFound))
			})
		})
		Context("with missing Content-Type header", func() {
			It("should return 415", func() {
				request.Header.Del("Content-Type")
				handler.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(http.StatusUnsupportedMediaType))
			})
		})
		Context("with invalid JSON", func() {
			It("should return 400", func() {
				request.Body = ioutil.NopCloser(strings.NewReader("test"))
				handler.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(http.StatusBadRequest))
			})
		})
		Context("with missing name field", func() {
			It("should return 400", func() {
				request.Body = marshalToJSON(payload{Email: "test@test.com"})
				handler.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(http.StatusBadRequest))
			})
		})
		Context("with invalid email", func() {
			It("should return 400", func() {
				request.Body = marshalToJSON(payload{Name: "test", Email: "wrong"})
				handler.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(http.StatusBadRequest))
			})
		})
		Context("with valid JSON", func() {
			It("should return 200", func() {
				handler.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(http.StatusOK))
			})
			It("should return a json containing the right name and email", func() {
				handler.ServeHTTP(recorder, request)
				responseBody := payload{}
				json.NewDecoder(recorder.Body).Decode(&responseBody)
				Expect(recorder.Header().Get("Content-Type")).To(Equal(rest.MIME_JSON))
				Expect(responseBody).To(Equal(payload{Name: "test", Email: "test@test.com"}))
			})
		})
		Context("with valid YAML", func() {
			It("should accept it and return it as YAML", func() {
				request.Header.Set("Content-Type", rest.MIME_YAML)
				request.Header.Set("Accept", rest.MIME_YAML)
				request.Body = marshalToYAML(&payload{Name: "test", Email: "test@test.com"})
				handler.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(http.StatusOK))
				Expect(recorder.Header().Get("Content-Type")).To(Equal(rest.MIME_YAML))
				responseBody := payload{}
				yaml.Unmarshal(recorder.Body.Bytes(), &responseBody)
				Expect(responseBody).To(Equal(payload{Name: "test", Email: "test@test.com"}))
			})
		})
	})
})
