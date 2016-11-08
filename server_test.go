package mpdsub

import (
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestServerClose(t *testing.T) {
	pingC := make(chan struct{}, 0)
	db := &memoryDatabase{
		pingC: pingC,
	}

	s := newServer(db, nil, &Config{
		Keepalive: 10 * time.Millisecond,
	})
	for i := 0; i < 3; i++ {
		<-pingC
	}
	s.Close()
	close(pingC)
}

func TestServerServeHTTP(t *testing.T) {
	tests := []struct {
		name string
		db   database
		cfg  *Config

		method string
		target string
		values url.Values

		httpCode int
		status   string
		code     int
	}{
		{
			name: "bad HTTP method",

			method: http.MethodPut,

			httpCode: http.StatusMethodNotAllowed,
		},
		{
			name: "missing username",

			code:   codeMissingParameter,
			status: statusFailed,
		},
		{
			name: "missing client",

			values: url.Values{
				"u": []string{"test"},
				"p": []string{"test"},
			},

			code:   codeMissingParameter,
			status: statusFailed,
		},
		{
			name: "missing version",

			values: url.Values{
				"u": []string{"test"},
				"c": []string{"test"},
			},

			code:   codeMissingParameter,
			status: statusFailed,
		},
		{
			name: "missing password and token",

			values: url.Values{
				"u": []string{"test"},
				"c": []string{"test"},
				"v": []string{"1.14.0"},
			},

			code:   codeMissingParameter,
			status: statusFailed,
		},
		{
			name: "missing salt",

			values: url.Values{
				"u": []string{"test"},
				"t": []string{"test"},
				"c": []string{"test"},
				"v": []string{"1.14.0"},
			},

			code:   codeMissingParameter,
			status: statusFailed,
		},
		{
			name: "incorrect username",
			cfg: &Config{
				SubsonicUser:     "test",
				SubsonicPassword: "test",
			},

			values: url.Values{
				"u": []string{"foo"},
				"p": []string{"test"},
				"c": []string{"test"},
				"v": []string{"1.14.0"},
			},

			code:   codeUnauthorized,
			status: statusFailed,
		},
		{
			name: "incorrect password",
			cfg: &Config{
				SubsonicUser:     "test",
				SubsonicPassword: "test",
			},

			values: url.Values{
				"u": []string{"test"},
				"p": []string{"foo"},
				"c": []string{"test"},
				"v": []string{"1.14.0"},
			},

			code:   codeUnauthorized,
			status: statusFailed,
		},
		{
			name: "incorrect encoded password",
			cfg: &Config{
				SubsonicUser:     "test",
				SubsonicPassword: "test",
			},

			values: url.Values{
				"u": []string{"test"},
				"p": []string{"enc:00"},
				"c": []string{"test"},
				"v": []string{"1.14.0"},
			},

			code:   codeUnauthorized,
			status: statusFailed,
		},
		{
			name: "OK password",
			cfg: &Config{
				SubsonicUser:     "test",
				SubsonicPassword: "test",
			},

			method: http.MethodGet,
			target: "/rest/ping.view",

			values: url.Values{
				"u": []string{"test"},
				"p": []string{"test"},
				"c": []string{"test"},
				"v": []string{"1.14.0"},
			},

			status: statusOK,
		},
		{
			name: "OK encoded password",
			cfg: &Config{
				SubsonicUser:     "test",
				SubsonicPassword: "test",
			},

			method: http.MethodGet,
			target: "/rest/ping.view",

			values: url.Values{
				"u": []string{"test"},
				"p": []string{"enc:74657374"},
				"c": []string{"test"},
				"v": []string{"1.14.0"},
			},

			status: statusOK,
		},
		{
			name: "OK token and salt",
			cfg: &Config{
				SubsonicUser:     "test",
				SubsonicPassword: "sesame",
			},

			method: http.MethodGet,
			target: "/rest/ping.view",

			values: url.Values{
				"u": []string{"test"},
				"t": []string{"26719a1196d2a940705a59634eb18eab"},
				"s": []string{"c19b2d"},
				"c": []string{"test"},
				"v": []string{"1.14.0"},
			},

			status: statusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withServer(t, tt.db, nil, tt.cfg, func(base string) {
				res := testRequest(t, base, tt.method, tt.target, tt.values)

				if tt.httpCode != 0 {
					if want, got := tt.httpCode, res.StatusCode; want != got {
						t.Fatalf("unexpected HTTP status code:\n- want: %03d\n-  got: %03d", want, got)
					}

					return
				}

				c := mustDecodeXML(t, res)

				if want, got := tt.status, c.Status; want != got {
					t.Fatalf("unexpected Status:\n- want: %q\n-  got: %q", want, got)
				}

				if c.Error != nil {
					if want, got := tt.code, c.Error.Code; want != got {
						t.Fatalf("unexpected error code:\n- want: %q\n-  got: %q", want, got)
					}
				}
			})
		})
	}
}
