package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/taskcluster/httpbackoff"
	"github.com/taskcluster/slugid-go/slugid"
	tcclient "github.com/taskcluster/taskcluster-client-go"
	tcurls "github.com/taskcluster/taskcluster-lib-urls"
)

var (
	rootURL         = os.Getenv("TASKCLUSTER_ROOT_URL")
	permCredentials = &tcclient.Credentials{
		ClientID:    os.Getenv("TASKCLUSTER_CLIENT_ID"),
		AccessToken: os.Getenv("TASKCLUSTER_ACCESS_TOKEN"),
	}
)

func newTestClient() *httpbackoff.Client {
	return &httpbackoff.Client{
		BackOffSettings: &backoff.ExponentialBackOff{
			InitialInterval:     1 * time.Millisecond,
			RandomizationFactor: 0.2,
			Multiplier:          1.2,
			MaxInterval:         5 * time.Millisecond,
			MaxElapsedTime:      20 * time.Millisecond,
			Clock:               backoff.SystemClock,
		},
	}
}

type IntegrationTest func(t *testing.T, creds *tcclient.Credentials) *httptest.ResponseRecorder

func skipIfNoPermCreds(t *testing.T) {
	if rootURL == "" {
		t.Skip("TASKCLUSTER_ROOT_URL not set - skipping test")
	}
	if permCredentials.ClientID == "" {
		t.Skip("TASKCLUSTER_CLIENT_ID not set - skipping test")
	}
	if permCredentials.AccessToken == "" {
		t.Skip("TASKCLUSTER_ACCESS_TOKEN not set - skipping test")
	}
}

func testWithPermCreds(t *testing.T, test IntegrationTest, expectedStatusCode int) {
	skipIfNoPermCreds(t)
	res := test(t, permCredentials)
	checkStatusCode(
		t,
		res,
		expectedStatusCode,
	)
	checkHeaders(
		t,
		res,
		map[string]string{
			"X-Taskcluster-Proxy-Version":       version,
			"X-Taskcluster-Proxy-Revision":      revision,
			"X-Taskcluster-Proxy-Perm-ClientId": permCredentials.ClientID,
			// N.B. the http library does not distinguish between header entries
			// that have an empty "" value, and non-existing entries
			"X-Taskcluster-Proxy-Temp-ClientId": "",
			"X-Taskcluster-Proxy-Temp-Scopes":   "",
		},
	)
}

func testWithTempCreds(t *testing.T, test IntegrationTest, expectedStatusCode int) {
	skipIfNoPermCreds(t)
	tempScopes := []string{
		"assume:project:taskcluster:taskcluster-proxy-tester",
	}

	tempScopesBytes, err := json.Marshal(tempScopes)
	if err != nil {
		t.Fatal("Bug in test")
	}
	tempScopesJSON := string(tempScopesBytes)

	tempCredsClientID := "garbage/" + slugid.Nice()
	tempCredentials, err := permCredentials.CreateNamedTemporaryCredentials(tempCredsClientID, 1*time.Hour, tempScopes...)
	if err != nil {
		t.Fatalf("Could not generate temp credentials")
	}
	res := test(t, tempCredentials)
	checkStatusCode(
		t,
		res,
		expectedStatusCode,
	)
	checkHeaders(
		t,
		res,
		map[string]string{
			"X-Taskcluster-Proxy-Version":       version,
			"X-Taskcluster-Proxy-Revision":      revision,
			"X-Taskcluster-Proxy-Temp-ClientId": tempCredsClientID,
			"X-Taskcluster-Proxy-Temp-Scopes":   tempScopesJSON,
			// N.B. the http library does not distinguish between header entries
			// that have an empty "" value, and non-existing entries
			"X-Taskcluster-Proxy-Perm-ClientId": "",
		},
	)
}

func checkHeaders(t *testing.T, res *httptest.ResponseRecorder, requiredHeaders map[string]string) {
	for headerKey, expectedHeaderValue := range requiredHeaders {
		actualHeaderValue := res.Header().Get(headerKey)
		if actualHeaderValue != expectedHeaderValue {
			// N.B. the http library does not distinguish between header
			// entries that have an empty "" value, and non-existing entries
			if expectedHeaderValue != "" {
				t.Errorf("Expected header %q to be %q but it was %q", headerKey, expectedHeaderValue, actualHeaderValue)
				t.Logf("Full headers: %q", res.Header())
			} else {
				t.Errorf("Expected header %q to not be present, or to be an empty string (\"\"), but it was %q", headerKey, actualHeaderValue)
			}
		}
	}
}

func checkStatusCode(t *testing.T, res *httptest.ResponseRecorder, statusCode int) {
	respBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("Could not read response body: %v", err)
	}
	// Make sure we get at least a few bytes of a response body...
	// Even HTTP 303 should have some body, see
	// https://tools.ietf.org/html/rfc7231#section-6.4.4
	// TestRetrievePrivateArtifact retrieves an artifact with
	// 14 bytes, so let's set that as minimum.
	if len(respBody) < 14 {
		t.Error("Expected a response body (at least 14 bytes), but get less (or none).")
		t.Logf("Headers: %s", res.Header())
		t.Logf("Response received:\n%s", string(respBody))
	}
	if res.Code != statusCode {
		t.Errorf("Expected status code %v but got %v", statusCode, res.Code)
		t.Logf("Headers: %s", res.Header())
		t.Logf("Response received:\n%s", string(respBody))
	}
}

func TestBewit(t *testing.T) {
	taskID, artifactName, artifactContent := createPrivateArtifact(t, nil)
	test := func(t *testing.T, creds *tcclient.Credentials) *httptest.ResponseRecorder {

		// Test setup
		routes := NewRoutes(
			rootURL,
			tcclient.Client{
				Credentials: creds,
			},
		)
		// go identifier `url` is already used by net/url package - so call var `_url` instead
		_url := tcurls.API(os.Getenv("TASKCLUSTER_ROOT_URL"), "queue", "v1", "task/"+taskID+"/runs/0/artifacts/"+url.QueryEscape(artifactName))
		req, err := http.NewRequest(
			"POST",
			"http://localhost:60024/bewit",
			bytes.NewBufferString(_url),
		)
		if err != nil {
			log.Fatal(err)
		}
		res := httptest.NewRecorder()

		// Function to test
		routes.BewitHandler(res, req)

		// Validate results
		bewitURLFromLocation := res.Header().Get("Location")
		bewitURLFromResponseBody := res.Body.String()
		if bewitURLFromLocation != bewitURLFromResponseBody {
			t.Fatalf("Got inconsistent results between Location header (%v) and Response body (%v).", bewitURLFromLocation, bewitURLFromResponseBody)
		}
		_, err = url.Parse(bewitURLFromLocation)
		if err != nil {
			t.Fatalf("Bewit URL returned is invalid: %q", bewitURLFromLocation)
		}
		resp, _, err := newTestClient().Get(bewitURLFromLocation)
		if err != nil {
			t.Fatalf("Exception thrown:\n%s", err)
		}
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Exception thrown:\n%s", err)
		}
		if string(respBody) != string(artifactContent) {
			t.Fatalf("Expected response body to be %v but was %v", artifactContent, respBody)
		}
		return res
	}
	testWithPermCreds(t, test, 303)
	testWithTempCreds(t, test, 303)
}

func TestAuthorizationDelegate(t *testing.T) {
	test := func(name string, scopes []string) IntegrationTest {
		return func(t *testing.T, creds *tcclient.Credentials) *httptest.ResponseRecorder {
			// Test setup
			routes := NewRoutes(
				rootURL,
				tcclient.Client{
					Authenticate: true,
					Credentials: &tcclient.Credentials{
						ClientID:         creds.ClientID,
						AccessToken:      creds.AccessToken,
						Certificate:      creds.Certificate,
						AuthorizedScopes: scopes,
					},
				},
			)

			// Requires scope "auth:azure-table:read-write:fakeaccount/DuMmYtAbLe"
			req, err := http.NewRequest(
				"GET",
				fmt.Sprintf(
					"http://localhost:60024/api/auth/v1/azure/%s/table/%s/read-write",
					"fakeaccount",
					"DuMmYtAbLe",
				),
				// Note: we don't set body to nil as a server http request
				// cannot have a nil body. See:
				// https://golang.org/pkg/net/http/#Request
				new(bytes.Buffer),
			)
			if err != nil {
				log.Fatal(err)
			}
			res := httptest.NewRecorder()

			// Function to test
			routes.APIHandler(res, req)
			return res
		}
	}
	testWithPermCreds(t, test("A", []string{"auth:azure-table:read-write:fakeaccount/DuMmYtAbLe"}), 404)
	testWithTempCreds(t, test("B", []string{"auth:azure-table:read-write:fakeaccount/DuMmYtAbLe"}), 404)
	testWithPermCreds(t, test("C", []string{"queue:get-artifact:taskcluster-proxy-test/512-random-bytes"}), 403)
	testWithTempCreds(t, test("D", []string{"queue:get-artifact:taskcluster-proxy-test/512-random-bytes"}), 403)
}

func TestAPICallWithPayload(t *testing.T) {
	test := func(t *testing.T, creds *tcclient.Credentials) *httptest.ResponseRecorder {

		// Test setup
		routes := NewRoutes(
			rootURL,
			tcclient.Client{
				Authenticate: true,
				Credentials:  creds,
			},
		)
		taskID := slugid.Nice()
		taskGroupID := slugid.Nice()
		created := time.Now()
		deadline := created.AddDate(0, 0, 1)
		expires := deadline

		req, err := http.NewRequest(
			"POST",
			"http://localhost:60024/queue/v1/task/"+taskID+"/define",
			bytes.NewBufferString(
				`
{
  "provisionerId": "win-provisioner",
  "workerType": "win2008-worker",
  "schedulerId": "go-test-test-scheduler",
  "taskGroupId": "`+taskGroupID+`",
  "routes": [
    "garbage.dummy.route.12345",
    "garbage.dummy.route.54321"
  ],
  "priority": "high",
  "retries": 5,
  "created": "`+tcclient.Time(created).String()+`",
  "deadline": "`+tcclient.Time(deadline).String()+`",
  "expires": "`+tcclient.Time(expires).String()+`",
  "scopes": [
  ],
  "payload": {
    "features": {
      "relengApiProxy": true
    }
  },
  "metadata": {
    "description": "Stuff",
    "name": "[TC] Pete",
    "owner": "pmoore@mozilla.com",
    "source": "http://everywhere.com/"
  },
  "tags": {
    "createdForUser": "cbook@mozilla.com"
  },
  "extra": {
    "index": {
      "rank": 12345
    }
  }
}
`,
			),
		)
		if err != nil {
			log.Fatal(err)
		}
		res := httptest.NewRecorder()

		// Function to test
		routes.RootHandler(res, req)

		t.Logf("Created task https://queue.taskcluster.net/v1/task/%v", taskID)
		return res
	}
	testWithPermCreds(t, test, 200)
	testWithTempCreds(t, test, 200)
}

func TestNon200HasErrorBody(t *testing.T) {
	test := func(t *testing.T, creds *tcclient.Credentials) *httptest.ResponseRecorder {

		// Test setup
		routes := NewRoutes(
			rootURL,
			tcclient.Client{
				Authenticate: true,
				Credentials:  creds,
			},
		)
		taskID := slugid.Nice()

		req, err := http.NewRequest(
			"POST",
			"http://localhost:60024/queue/v1/task/"+taskID+"/define",
			bytes.NewBufferString(
				`{"comment": "Valid json so that we hit endpoint, but should not result in http 200"}`,
			),
		)
		if err != nil {
			log.Fatal(err)
		}
		res := httptest.NewRecorder()

		// Function to test
		routes.RootHandler(res, req)

		// Validate results
		return res

	}
	testWithPermCreds(t, test, 400)
	testWithTempCreds(t, test, 400)
}

func TestOversteppedScopes(t *testing.T) {
	test := func(t *testing.T, creds *tcclient.Credentials) *httptest.ResponseRecorder {

		// Test setup
		routes := NewRoutes(
			rootURL,
			tcclient.Client{
				Authenticate: true,
				Credentials:  creds,
			},
		)

		// This scope is not in the scopes of the temp credentials, which would
		// happen if a task declares a scope that the provisioner does not
		// grant.
		routes.Credentials.AuthorizedScopes = []string{"secrets:get:garbage/pmoore/foo"}

		req, err := http.NewRequest(
			"GET",
			"http://localhost:60024/secrets/v1/secret/garbage/pmoore/foo",
			new(bytes.Buffer),
		)
		if err != nil {
			log.Fatal(err)
		}
		res := httptest.NewRecorder()

		// Function to test
		routes.RootHandler(res, req)

		// Validate results
		checkHeaders(
			t,
			res,
			map[string]string{
				"X-Taskcluster-Endpoint":          tcurls.API(rootURL, "secrets", "v1", "secret/garbage/pmoore/foo"),
				"X-Taskcluster-Authorized-Scopes": `["secrets:get:garbage/pmoore/foo"]`,
			},
		)
		return res
	}
	testWithTempCreds(t, test, 401)
}

func TestBadCredsReturns500(t *testing.T) {
	routes := NewRoutes(
		rootURL,
		tcclient.Client{
			Authenticate: true,
			Credentials: &tcclient.Credentials{
				ClientID:    "abc",
				AccessToken: "def",
				Certificate: "ghi", // baaaad certificate
			},
		},
	)
	req, err := http.NewRequest(
		"GET",
		"http://localhost:60024/secrets/v1/secret/garbage/pmoore/foo",
		new(bytes.Buffer),
	)
	if err != nil {
		log.Fatal(err)
	}
	res := httptest.NewRecorder()

	// Function to test
	routes.RootHandler(res, req)
	// Validate results
	checkStatusCode(t, res, 500)
}

func TestInvalidEndpoint(t *testing.T) {
	test := func(t *testing.T, creds *tcclient.Credentials) *httptest.ResponseRecorder {

		// Test setup
		routes := NewRoutes(
			rootURL,
			tcclient.Client{
				Authenticate: true,
				Credentials:  creds,
			},
		)

		req, err := http.NewRequest(
			"GET",
			"http://localhost:60024/x@/", // invalid endpoint
			new(bytes.Buffer),
		)
		if err != nil {
			log.Fatal(err)
		}
		res := httptest.NewRecorder()

		// Function to test
		routes.RootHandler(res, req)

		// Validate results
		checkHeaders(
			t,
			res,
			map[string]string{
				"X-Taskcluster-Endpoint": "",
			},
		)
		return res
	}
	testWithTempCreds(t, test, 404)
	testWithPermCreds(t, test, 404)
}

func TestRetrievePrivateArtifact(t *testing.T) {
	taskID, artifactName, artifactContent := createPrivateArtifact(t, nil)
	test := func(t *testing.T, creds *tcclient.Credentials) *httptest.ResponseRecorder {

		// Test setup
		routes := NewRoutes(
			rootURL,
			tcclient.Client{
				Authenticate: true,
				Credentials:  creds,
			},
		)

		req, err := http.NewRequest(
			"GET",
			"http://localhost:60024/queue/v1/task/"+taskID+"/runs/0/artifacts/"+url.QueryEscape(artifactName),
			nil,
		)
		if err != nil {
			log.Fatal(err)
		}
		res := httptest.NewRecorder()

		// Function to test
		routes.RootHandler(res, req)

		if res.Body.String() != string(artifactContent) {
			t.Fatalf("Artifact content does not match: %v vs %v", res.Body.String(), string(artifactContent))
		}
		return res
	}
	testWithPermCreds(t, test, 200)
	testWithTempCreds(t, test, 200)
}
