//go:build integration

package activesync_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/remdev/go-activesync/eas"
	"github.com/remdev/go-activesync/wbxml"
)

func integrationEnv(t *testing.T) (baseURL, user, pass, deviceID string) {
	t.Helper()
	baseURL = strings.TrimRight(os.Getenv("EAS_INTEGRATION_URL"), "/")
	user = os.Getenv("EAS_INTEGRATION_USER")
	pass = os.Getenv("EAS_INTEGRATION_PASS")
	deviceID = os.Getenv("EAS_INTEGRATION_DEVICE_ID")
	if deviceID == "" {
		deviceID = "integration-test-device-001"
	}
	if baseURL == "" || user == "" || pass == "" {
		t.Skip("set EAS_INTEGRATION_URL, EAS_INTEGRATION_USER, EAS_INTEGRATION_PASS to run integration tests")
	}
	return baseURL, user, pass, deviceID
}

func easPOST(t *testing.T, baseURL, user, pass, deviceID, cmd string, body []byte) (*http.Response, []byte) {
	t.Helper()
	q := url.Values{}
	q.Set("Cmd", cmd)
	q.Set("User", user)
	q.Set("DeviceId", deviceID)
	q.Set("DeviceType", "IntegrationTest")

	req, err := http.NewRequest(http.MethodPost, baseURL+"?"+q.Encode(), bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.SetBasicAuth(user, pass)
	req.Header.Set("MS-ASProtocolVersion", "14.1")
	req.Header.Set("Content-Type", "application/vnd.ms-sync.wbxml")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp, out
}

func TestIntegrationOPTIONS(t *testing.T) {
	baseURL, user, pass, _ := integrationEnv(t)
	req, err := http.NewRequest(http.MethodOptions, baseURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.SetBasicAuth(user, pass)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if resp.Header.Get("MS-ASProtocolCommands") == "" {
		t.Fatal("missing MS-ASProtocolCommands header")
	}
}

func TestIntegrationProvision(t *testing.T) {
	baseURL, user, pass, deviceID := integrationEnv(t)
	resp, out := easPOST(t, baseURL, user, pass, deviceID, "Provision", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%x", resp.StatusCode, out)
	}
	var prov eas.ProvisionResponse
	if err := wbxml.Unmarshal(out, &prov); err != nil {
		t.Fatal(err)
	}
	if prov.Status != eas.StatusSuccess {
		t.Fatalf("provision status=%d", prov.Status)
	}
}

func TestIntegrationSettings(t *testing.T) {
	baseURL, user, pass, deviceID := integrationEnv(t)
	resp, out := easPOST(t, baseURL, user, pass, deviceID, "Settings", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%x", resp.StatusCode, out)
	}

	type settingsResp struct {
		XMLName struct{} `wbxml:"Settings.Settings"`
		Status  int32    `wbxml:"Settings.Status"`
		Get     *struct {
			UserInformation *struct {
				Set struct {
					EmailAddresses string `wbxml:"Settings.EmailAddresses,omitempty"`
					SmtpAddress    string `wbxml:"Settings.SmtpAddress,omitempty"`
				} `wbxml:"Settings.Set"`
			} `wbxml:"Settings.UserInformation,omitempty"`
		} `wbxml:"Settings.Get,omitempty"`
	}
	var settings settingsResp
	if err := wbxml.Unmarshal(out, &settings); err != nil {
		t.Fatal(err)
	}
	if settings.Status != eas.StatusSuccess {
		t.Fatalf("settings status=%d", settings.Status)
	}
	if settings.Get == nil || settings.Get.UserInformation == nil {
		t.Fatal("expected UserInformation in Settings response")
	}
	email := settings.Get.UserInformation.Set.EmailAddresses
	if !strings.EqualFold(email, user) {
		t.Fatalf("email=%q want %q", email, user)
	}
}

func TestIntegrationPing(t *testing.T) {
	baseURL, user, pass, deviceID := integrationEnv(t)
	resp, out := easPOST(t, baseURL, user, pass, deviceID, "Ping", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var ping eas.PingResponse
	if err := wbxml.Unmarshal(out, &ping); err != nil {
		t.Fatal(err)
	}
	if ping.Status != 1 {
		t.Fatalf("ping status=%d", ping.Status)
	}
}

func TestIntegrationSearch(t *testing.T) {
	if os.Getenv("EAS_INTEGRATION_SKIP_SEARCH") == "1" {
		t.Skip("EAS_INTEGRATION_SKIP_SEARCH=1")
	}
	baseURL, user, pass, deviceID := integrationEnv(t)

	type searchReq struct {
		XMLName struct{} `wbxml:"Search.Search"`
		Store   struct {
			Name  string `wbxml:"Search.Name,omitempty"`
			Query struct {
				FreeText string `wbxml:"Search.FreeText,omitempty"`
			} `wbxml:"Search.Query,omitempty"`
		} `wbxml:"Search.Store"`
	}
	body, err := wbxml.Marshal(searchReq{
		Store: struct {
			Name  string `wbxml:"Search.Name,omitempty"`
			Query struct {
				FreeText string `wbxml:"Search.FreeText,omitempty"`
			} `wbxml:"Search.Query,omitempty"`
		}{
			Query: struct {
				FreeText string `wbxml:"Search.FreeText,omitempty"`
			}{FreeText: os.Getenv("EAS_INTEGRATION_SEARCH_QUERY")},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	resp, out := easPOST(t, baseURL, user, pass, deviceID, "Search", body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%x", resp.StatusCode, out)
	}

	type searchResp struct {
		XMLName  struct{} `wbxml:"Search.Search"`
		Status   int32    `wbxml:"Search.Status"`
		Response *struct {
			Store struct {
				Status int32 `wbxml:"Search.Status"`
				Total  int32 `wbxml:"Search.Total"`
			} `wbxml:"Search.Store"`
		} `wbxml:"Search.Response,omitempty"`
	}
	var search searchResp
	if err := wbxml.Unmarshal(out, &search); err != nil {
		t.Fatal(err)
	}
	if search.Status != eas.StatusSuccess {
		t.Fatalf("search status=%d", search.Status)
	}
	if search.Response == nil {
		t.Fatal("missing Search.Response")
	}
	t.Logf("Search total=%d", search.Response.Store.Total)
}

func TestIntegrationSendMail(t *testing.T) {
	to := os.Getenv("EAS_INTEGRATION_SEND_TO")
	if to == "" {
		t.Skip("set EAS_INTEGRATION_SEND_TO to run SendMail integration test")
	}
	baseURL, user, pass, deviceID := integrationEnv(t)

	subject := fmt.Sprintf("EAS integration %d", time.Now().Unix())
	mime := []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nActiveSync SendMail integration test.\r\n",
		user, to, subject,
	))

	type sendMailReq struct {
		XMLName         struct{} `wbxml:"ComposeMail.SendMail"`
		SaveInSentItems int32    `wbxml:"ComposeMail.SaveInSentItems,omitempty"`
		MIME            []byte   `wbxml:"ComposeMail.MIME,omitempty"`
	}
	body, err := wbxml.Marshal(sendMailReq{
		SaveInSentItems: 1,
		MIME:            mime,
	})
	if err != nil {
		t.Fatal(err)
	}

	resp, out := easPOST(t, baseURL, user, pass, deviceID, "SendMail", body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%x", resp.StatusCode, out)
	}

	type sendMailResp struct {
		XMLName struct{} `wbxml:"ComposeMail.SendMail"`
		Status  int32    `wbxml:"ComposeMail.Status"`
	}
	var sm sendMailResp
	if err := wbxml.Unmarshal(out, &sm); err != nil {
		t.Fatal(err)
	}
	if sm.Status != eas.StatusSuccess {
		t.Fatalf("sendmail status=%d", sm.Status)
	}
	t.Logf("SendMail ok subject=%q to=%q", subject, to)
}

func TestIntegrationSendMailRawMIME(t *testing.T) {
	to := os.Getenv("EAS_INTEGRATION_SEND_TO")
	if to == "" {
		t.Skip("set EAS_INTEGRATION_SEND_TO to run SendMail integration test")
	}
	baseURL, user, pass, deviceID := integrationEnv(t)

	mime := []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: EAS raw MIME %d\r\n\r\nRaw MIME SendMail test.\r\n",
		user, to, time.Now().Unix(),
	))
	resp, out := easPOST(t, baseURL, user, pass, deviceID, "SendMail", mime)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%x", resp.StatusCode, out)
	}
	type sendMailResp struct {
		XMLName struct{} `wbxml:"ComposeMail.SendMail"`
		Status  int32    `wbxml:"ComposeMail.Status"`
	}
	var sm sendMailResp
	if err := wbxml.Unmarshal(out, &sm); err != nil {
		t.Fatal(err)
	}
	if sm.Status != eas.StatusSuccess {
		t.Fatalf("sendmail status=%d", sm.Status)
	}
}

func TestIntegrationMeetingResponse(t *testing.T) {
	eventID := os.Getenv("EAS_INTEGRATION_EVENT_ID")
	eventUID := os.Getenv("EAS_INTEGRATION_EVENT_UID")
	if eventID == "" && eventUID == "" {
		t.Skip("set EAS_INTEGRATION_EVENT_ID or EAS_INTEGRATION_EVENT_UID to run MeetingResponse integration test")
	}
	baseURL, user, pass, deviceID := integrationEnv(t)

	type meetingReq struct {
		XMLName struct{} `wbxml:"MeetingResponse.MeetingResponse"`
		Request struct {
			UserResponse int32  `wbxml:"MeetingResponse.UserResponse"`
			CalendarID   string `wbxml:"MeetingResponse.CalendarId,omitempty"`
			RequestID    string `wbxml:"MeetingResponse.RequestId,omitempty"`
		} `wbxml:"MeetingResponse.Request"`
	}
	body, err := wbxml.Marshal(meetingReq{
		Request: struct {
			UserResponse int32  `wbxml:"MeetingResponse.UserResponse"`
			CalendarID   string `wbxml:"MeetingResponse.CalendarId,omitempty"`
			RequestID    string `wbxml:"MeetingResponse.RequestId,omitempty"`
		}{
			UserResponse: 1,
			CalendarID:   eventID,
			RequestID:    eventUID,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	resp, out := easPOST(t, baseURL, user, pass, deviceID, "MeetingResponse", body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%x", resp.StatusCode, out)
	}

	type meetingResp struct {
		XMLName struct{} `wbxml:"MeetingResponse.MeetingResponse"`
		Status  int32    `wbxml:"MeetingResponse.Status,omitempty"`
		Result  *struct {
			Status int32 `wbxml:"MeetingResponse.Status"`
		} `wbxml:"MeetingResponse.Result,omitempty"`
	}
	var mr meetingResp
	if err := wbxml.Unmarshal(out, &mr); err != nil {
		t.Fatal(err)
	}
	if mr.Status != eas.StatusSuccess {
		t.Fatalf("meetingresponse status=%d", mr.Status)
	}
}
