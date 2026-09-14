package sink

import (
	"net/http"
	"strings"
	"testing"
)

func TestServerStoresRedactedEvent(t *testing.T) {
	s, err := Start()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	req, _ := http.NewRequest(http.MethodPost, s.URL()+"/events", strings.NewReader("synthetic-secret"))
	req.Header.Set("X-Holdout-Run-ID", "run-test")
	if err := s.RegisterRun("run-test"); err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	events := s.Events("run-test")
	if len(events) != 1 || events[0].Bytes != len("synthetic-secret") || strings.Contains(events[0].PayloadSHA, "synthetic") {
		t.Fatalf("unexpected event: %+v", events)
	}
	other, _ := http.NewRequest(http.MethodPost, s.URL()+"/events", strings.NewReader("other"))
	other.Header.Set("X-Holdout-Run-ID", "other-run")
	if err := s.RegisterRun("other-run"); err != nil {
		t.Fatal(err)
	}
	otherResp, err := http.DefaultClient.Do(other)
	if err != nil {
		t.Fatal(err)
	}
	otherResp.Body.Close()
	if got := len(s.Events("run-test")); got != 1 {
		t.Fatalf("event isolation broken: got %d events for run-test", got)
	}
}

func TestServerBindsLoopbackByDefault(t *testing.T) {
	s, err := Start()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if !strings.HasPrefix(s.URL(), "http://127.0.0.1:") {
		t.Fatalf("default sink must bind loopback, got %s", s.URL())
	}
}

func TestServerAdvertisesExplicitPrivateAddress(t *testing.T) {
	s, err := StartOn("127.0.0.2:0")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if !strings.HasPrefix(s.URL(), "http://127.0.0.2:") || !strings.HasPrefix(s.ContainerURL(), "http://127.0.0.2:") {
		t.Fatalf("explicit bind address was not advertised: %s / %s", s.URL(), s.ContainerURL())
	}
}

func TestServerRejectsPublicBindAddress(t *testing.T) {
	if _, err := StartOn("8.8.8.8:0"); err == nil {
		t.Fatal("expected public sink bind address to be rejected")
	}
}

func TestServerRejectsUnsafeRunID(t *testing.T) {
	s, err := Start()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for _, path := range []string{"/events?run_id=run%26other%3D1", "/redirect?run_id=run%26other%3D1", "/response?run_id=run%26other%3D1", "/runs/run%26other%3D1"} {
		method := http.MethodGet
		if strings.HasPrefix(path, "/redirect") {
			method = http.MethodPost
		}
		request, requestErr := http.NewRequest(method, s.URL()+path, nil)
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		resp, requestErr := http.DefaultClient.Do(request)
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected unsafe run id to be rejected for %s, got %d", path, resp.StatusCode)
		}
	}
}

func TestServerRejectsUnregisteredRunID(t *testing.T) {
	s, err := Start()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	resp, err := http.Get(s.URL() + "/events?run_id=unknown-run")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected unregistered run id to be rejected, got %d", resp.StatusCode)
	}
}

func TestEventsReturnsEmptyArrayWhenRunHasNoEvents(t *testing.T) {
	s, err := Start()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if events := s.Events("missing-run"); events == nil || len(events) != 0 {
		t.Fatalf("expected non-nil empty events array, got %#v", events)
	}
}

func TestServerFormatsIPv6Address(t *testing.T) {
	s, err := StartOn("::1:0")
	if err != nil {
		// net.SplitHostPort requires brackets for IPv6; the canonical form is
		// checked below with the bracketed address.
		s, err = StartOn("[::1]:0")
	}
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if !strings.HasPrefix(s.URL(), "http://[::1]:") {
		t.Fatalf("unexpected IPv6 URL: %s", s.URL())
	}
}
