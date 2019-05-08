package route

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/webhook/statefulset"
	"github.com/pingcap/tidb-operator/pkg/webhook/util"
	"k8s.io/api/admission/v1beta1"
)

// admitFunc is the type we use for all of our validators
type admitFunc func(v1beta1.AdmissionReview) *v1beta1.AdmissionResponse

// serve handles the http portion of a request prior to handing to an admit
// function
func serve(w http.ResponseWriter, r *http.Request, admit admitFunc) {

	var body []byte
	var contentType string
	responseAdmissionReview := v1beta1.AdmissionReview{}
	requestedAdmissionReview := v1beta1.AdmissionReview{}
	deserializer := util.GetCodec()

	// The AdmissionReview that will be returned
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		} else {
			responseAdmissionReview.Response = util.ToAdmissionResponse(err)
			goto returnData
		}
	} else {
		err := errors.New("request body is nil!")
		responseAdmissionReview.Response = util.ToAdmissionResponse(err)
		goto returnData
	}

	// verify the content type is accurate
	contentType = r.Header.Get("Content-Type")
	if contentType != "application/json" {
		err := errors.New("expect application/json")
		responseAdmissionReview.Response = util.ToAdmissionResponse(err)
		goto returnData
	}

	// The AdmissionReview that was sent to the webhook
	if _, _, err := deserializer.Decode(body, nil, &requestedAdmissionReview); err != nil {
		responseAdmissionReview.Response = util.ToAdmissionResponse(err)
	} else {
		// pass to admitFunc
		responseAdmissionReview.Response = admit(requestedAdmissionReview)
	}

	// Return the same UID
	responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID

returnData:
	respBytes, err := json.Marshal(responseAdmissionReview)
	if err != nil {
		glog.Errorf("%v", err)
	}
	if _, err := w.Write(respBytes); err != nil {
		glog.Errorf("%v", err)
	}
}

func ServeStatefulSets(w http.ResponseWriter, r *http.Request) {
	serve(w, r, statefulset.AdmitStatefulSets)
}
