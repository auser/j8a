package content

import (
	"bytes"
	"github.com/klauspost/compress/flate"
	"github.com/simonmittag/j8a"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

func TestContentEncodingPermutationsOnProxyHandlerUpstreamNoContentEncoding(t *testing.T) {
	tests := map[string]struct {
		reqUrlSlug                      string
		reqAcceptEncodingHeader         string
		reqSendAcceptEncodingHeader     bool
		wantResStatusCode               int
		wantResContentEncodingHeader    string
		wantResVaryAcceptEncodingHeader bool
		wantResBodyContent              string
	}{
		"noAcceptEncodingSendsIdentity": {"/mse6/nocontentenc",
			"",
			false,
			200,
			"identity",
			false,
			"nocontentenc",
		},
		"emptyAcceptEncodingSendsIdentity": {"/mse6/nocontentenc",
			"",
			true,
			200,
			"identity",
			false,
			"nocontentenc",
		},
		//we assume upstream is identity if it's not providing a content encoding header
		"starAcceptEncodingSendsGzip": {"/mse6/nocontentenc",
			"*",
			true,
			200,
			"gzip",
			false,
			"nocontentenc",
		},
		"starCommaGzipAcceptEncodingSendsGzip": {"/mse6/nocontentenc",
			"*,gzip",
			true,
			200,
			"gzip",
			false,
			"nocontentenc",
		},
		"identityCommaGzipAcceptEncodingSendsGzip": {"/mse6/nocontentenc",
			"identity,gzip",
			true,
			200,
			"gzip",
			false,
			"nocontentenc",
		},
		"gzipCommaIdentityAcceptEncodingSendsGzip": {"/mse6/nocontentenc",
			"gzip,identity",
			true,
			200,
			"gzip",
			false,
			"nocontentenc",
		},
		"gzipCommaStarAcceptEncodingSendsGzip": {"/mse6/nocontentenc",
			"gzip,*",
			true,
			200,
			"gzip",
			false,
			"nocontentenc",
		},
		"gzipAcceptEncodingSendsGzip": {"/mse6/nocontentenc",
			"gzip",
			true,
			200,
			"gzip",
			false,
			"nocontentenc",
		},
		"gzipCommaUnknownAcceptEncodingSendsGzip": {"/mse6/nocontentenc",
			"gzip, unknown, moreunknown",
			true,
			200,
			"gzip",
			false,
			"nocontentenc",
		},
		"gzipCommaBrotliAcceptEncodingSendsGzip": {"/mse6/nocontentenc",
			"gzip, br",
			true,
			200,
			"gzip",
			false,
			"nocontentenc",
		},
		"brotliAcceptEncodingSendsBrotli": {"/mse6/nocontentenc",
			"br",
			true,
			200,
			"br",
			false,
			"nocontentenc",
		},
		"brotliCommaGzipAcceptEncodingSendsGzip": {"/mse6/nocontentenc",
			"br,gzip",
			true,
			200,
			"gzip",
			false,
			"nocontentenc",
		},
		"brotliCommaIdentityAcceptEncodingSendsBrotli": {"/mse6/nocontentenc",
			"br,identity",
			true,
			200,
			"br",
			false,
			"nocontentenc",
		},
		"brotliCommaStarAcceptEncodingSendsGzip": {"/mse6/nocontentenc",
			"br,*",
			true,
			200,
			"gzip",
			false,
			"nocontentenc",
		},
		"deflateAcceptEncodingSends406ResponseCode": {"/mse6/nocontentenc",
			"deflate",
			true,
			406,
			"identity",
			false,
			"406",
		},
		"unknownAcceptEncodingSends406ResponseCode": {"/mse6/nocontentenc",
			"unknown",
			true,
			406,
			"identity",
			false,
			"406",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			DownstreamContentEncodingFullIntegrity(tc.reqUrlSlug, tc.reqAcceptEncodingHeader, tc.reqSendAcceptEncodingHeader,
				tc.wantResContentEncodingHeader, tc.wantResVaryAcceptEncodingHeader, tc.wantResBodyContent, tc.wantResStatusCode, t)
		})
	}
}

func TestContentEncodingPermutationsOnProxyHandlerUpstreamUnknownContentEncoding(t *testing.T) {
	tests := map[string]struct {
		reqUrlSlug                      string
		reqAcceptEncodingHeader         string
		reqSendAcceptEncodingHeader     bool
		wantResStatusCode               int
		wantResContentEncodingHeader    string
		wantResVaryAcceptEncodingHeader bool
		wantResBodyContent              string
	}{
		"noAcceptEncodingSendsEncodedWithVary": {"/mse6/unknowncontentenc",
			"",
			false,
			200,
			"unknown",
			true,
			"unknowncontentenc",
		},
		"emptyAcceptEncodingSendsEncodedWithVary": {"/mse6/unknowncontentenc",
			"",
			true,
			200,
			"unknown",
			true,
			"unknowncontentenc",
		},
		"starAcceptEncodingSendsEncoded": {"/mse6/unknowncontentenc",
			"*",
			true,
			200,
			"unknown",
			false,
			"unknowncontentenc",
		},
		"starCommaGzipAcceptEncodingSendsEncoded": {"/mse6/unknowncontentenc",
			"*,gzip",
			true,
			200,
			"unknown",
			false,
			"unknowncontentenc",
		},
		"identityCommaGzipAcceptEncodingSendsEncodedWithVary": {"/mse6/unknowncontentenc",
			"identity,gzip",
			true,
			200,
			"unknown",
			true,
			"unknowncontentenc",
		},
		"gzipCommaIdentityAcceptEncodingSendsEncodedWithVary": {"/mse6/unknowncontentenc",
			"gzip,identity",
			true,
			200,
			"unknown",
			true,
			"unknowncontentenc",
		},
		"gzipCommaStarAcceptEncodingSendsEncoded": {"/mse6/unknowncontentenc",
			"gzip,*",
			true,
			200,
			"unknown",
			false,
			"unknowncontentenc",
		},
		"gzipAcceptEncodingSendsEncodedWithVary": {"/mse6/unknowncontentenc",
			"gzip",
			true,
			200,
			"unknown",
			true,
			"unknowncontentenc",
		},
		//this is cool. You need to ASK for at least one content encoding we understand but you still don't get a vary header if
		//the server returns unknown content enc cause he matches it against your, albeit incompatible, expectations
		"gzipCommaUnknownAcceptEncodingSendsEncoded": {"/mse6/unknowncontentenc",
			"gzip, unknown, moreunknown",
			true,
			200,
			"unknown",
			false,
			"unknowncontentenc",
		},
		"gzipCommaBrotliAcceptEncodingSendsEncodedWithVary": {"/mse6/unknowncontentenc",
			"gzip, br",
			true,
			200,
			"unknown",
			true,
			"unknowncontentenc",
		},
		"brotliAcceptEncodingSendsEncodedWithVary": {"/mse6/unknowncontentenc",
			"br",
			true,
			200,
			"unknown",
			true,
			"unknowncontentenc",
		},
		"brotliCommaGzipAcceptEncodingSendsEncodedWithVary": {"/mse6/unknowncontentenc",
			"br,gzip",
			true,
			200,
			"unknown",
			true,
			"unknowncontentenc",
		},
		"brotliCommaIdentityAcceptEncodingSendsEncodedWithVary": {"/mse6/unknowncontentenc",
			"br,identity",
			true,
			200,
			"unknown",
			true,
			"unknowncontentenc",
		},
		"brotliCommaStarAcceptEncodingSendsEncoded": {"/mse6/unknowncontentenc",
			"br,*",
			true,
			200,
			"unknown",
			false,
			"unknowncontentenc",
		},
		"deflateAcceptEncodingSends406ResponseCode": {"/mse6/unknowncontentenc",
			"deflate",
			true,
			406,
			"identity",
			false,
			"406",
		},
		//we don't allow asking only for server incompatible Accept-Encoding
		"unknownAcceptEncodingSends406ResponseCode": {"/mse6/unknowncontentenc",
			"unknown",
			true,
			406,
			"identity",
			false,
			"406",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			DownstreamContentEncodingFullIntegrity(tc.reqUrlSlug, tc.reqAcceptEncodingHeader, tc.reqSendAcceptEncodingHeader,
				tc.wantResContentEncodingHeader, tc.wantResVaryAcceptEncodingHeader, tc.wantResBodyContent, tc.wantResStatusCode, t)
		})
	}
}

func TestContentEncodingPermutationsOnProxyHandlerUpstreamIdentity(t *testing.T) {
	tests := map[string]struct {
		reqUrlSlug                      string
		reqAcceptEncodingHeader         string
		reqSendAcceptEncodingHeader     bool
		wantResStatusCode               int
		wantResContentEncodingHeader    string
		wantResVaryAcceptEncodingHeader bool
		wantResBodyContent              string
	}{
		"noAcceptEncodingSendsIdentity": {"/mse6/get",
			"",
			false,
			200,
			"identity",
			false,
			"get",
		},
		"emptyAcceptEncodingSendsIdentity": {"/mse6/get",
			"",
			true,
			200,
			"identity",
			false,
			"get",
		},
		"starAcceptEncodingSendsGzip": {"/mse6/get",
			"*",
			true,
			200,
			"gzip",
			false,
			"get",
		},
		"starCommaGzipAcceptEncodingSendsGzip": {"/mse6/get",
			"*,gzip",
			true,
			200,
			"gzip",
			false,
			"get",
		},
		"identityCommaGzipAcceptEncodingSendsGzip": {"/mse6/get",
			"identity,gzip",
			true,
			200,
			"gzip",
			false,
			"get",
		},
		"gzipCommaIdentityAcceptEncodingSendsGzip": {"/mse6/get",
			"gzip,identity",
			true,
			200,
			"gzip",
			false,
			"get",
		},
		"gzipCommaStarAcceptEncodingSendsGzip": {"/mse6/get",
			"gzip,*",
			true,
			200,
			"gzip",
			false,
			"get",
		},
		"gzipAcceptEncodingSendsGzip": {"/mse6/get",
			"gzip",
			true,
			200,
			"gzip",
			false,
			"get",
		},
		"gzipCommaUnknownAcceptEncodingSendsGzip": {"/mse6/get",
			"gzip, unknown, moreunknown",
			true,
			200,
			"gzip",
			false,
			"get",
		},
		"gzipCommaBrotliAcceptEncodingSendsGzip": {"/mse6/get",
			"gzip, br",
			true,
			200,
			"gzip",
			false,
			"get",
		},
		"brotliAcceptEncodingSendsBrotli": {"/mse6/get",
			"br",
			true,
			200,
			"br",
			false,
			"get",
		},
		"brotliCommaGzipAcceptEncodingSendsGzip": {"/mse6/get",
			"br,gzip",
			true,
			200,
			"gzip",
			false,
			"get",
		},
		"brotliCommaIdentityAcceptEncodingSendsBrotli": {"/mse6/get",
			"br,identity",
			true,
			200,
			"br",
			false,
			"get",
		},
		"brotliCommaStarAcceptEncodingSendsGzip": {"/mse6/get",
			"br,*",
			true,
			200,
			"gzip",
			false,
			"get",
		},
		"deflateAcceptEncodingSends406ResponseCode": {"/mse6/get",
			"deflate",
			true,
			406,
			"identity",
			false,
			"406",
		},
		"unknownAcceptEncodingSends406ResponseCode": {"/mse6/get",
			"unknown",
			true,
			406,
			"identity",
			false,
			"406",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			DownstreamContentEncodingFullIntegrity(tc.reqUrlSlug, tc.reqAcceptEncodingHeader, tc.reqSendAcceptEncodingHeader,
				tc.wantResContentEncodingHeader, tc.wantResVaryAcceptEncodingHeader, tc.wantResBodyContent, tc.wantResStatusCode, t)
		})
	}
}

func TestContentEncodingPermutationsOnProxyHandlerUpstreamGzip(t *testing.T) {
	tests := map[string]struct {
		reqUrlSlug                      string
		reqAcceptEncodingHeader         string
		reqSendAcceptEncodingHeader     bool
		wantResStatusCode               int
		wantResContentEncodingHeader    string
		wantResVaryAcceptEncodingHeader bool
		wantResBodyContent              string
	}{
		"noAcceptEncodingSendsEncoded": {"/mse6/gzip",
			"",
			false,
			200,
			"gzip",
			true,
			"gzip",
		},
		"emptyAcceptEncodingSendsEncoded": {"/mse6/gzip",
			"",
			true,
			200,
			"gzip",
			true,
			"gzip",
		},
		"starAcceptEncodingSendsEncoded": {"/mse6/gzip",
			"*",
			true,
			200,
			"gzip",
			false,
			"gzip",
		},
		"starCommaGzipAcceptEncodingSendsEncoded": {"/mse6/gzip",
			"*,gzip",
			true,
			200,
			"gzip",
			false,
			"gzip",
		},
		"identityCommaGzipAcceptEncodingSendsEncoded": {"/mse6/gzip",
			"identity,gzip",
			true,
			200,
			"gzip",
			false,
			"gzip",
		},
		"gzipCommaIdentityAcceptEncodingSendsEncoded": {"/mse6/gzip",
			"gzip,identity",
			true,
			200,
			"gzip",
			false,
			"gzip",
		},
		"gzipCommaStarAcceptEncodingSendsEncoded": {"/mse6/gzip",
			"gzip,*",
			true,
			200,
			"gzip",
			false,
			"gzip",
		},
		"gzipAcceptEncodingSendsEncoded": {"/mse6/gzip",
			"gzip",
			true,
			200,
			"gzip",
			false,
			"gzip",
		},
		"xgzipAcceptEncodingSendsEncodedNoVary": {"/mse6/gzip",
			"x-gzip",
			true,
			200,
			"gzip",
			false,
			"gzip",
		},
		"xGZIPAcceptEncodingSendsEncodedNoVary": {"/mse6/gzip",
			"x-GZIP",
			true,
			200,
			"gzip",
			false,
			"gzip",
		},
		"gzipCommaUnknownAcceptEncodingSendsEncoded": {"/mse6/gzip",
			"gzip, unknown, moreunknown",
			true,
			200,
			"gzip",
			false,
			"gzip",
		},
		"gzipCommaBrotliAcceptEncodingSendsGzip": {"/mse6/gzip",
			"gzip, br",
			true,
			200,
			"gzip",
			false,
			"gzip",
		},
		"brotliAcceptEncodingSendsGzipWithVary": {"/mse6/gzip",
			"br",
			true,
			200,
			"gzip",
			true,
			"gzip",
		},
		"brotliCommaGzipAcceptEncodingSendsGzip": {"/mse6/gzip",
			"br,gzip",
			true,
			200,
			"gzip",
			false,
			"gzip",
		},
		"brotliCommaIdentityAcceptEncodingSendsGzipWithVary": {"/mse6/gzip",
			"br,identity",
			true,
			200,
			"gzip",
			true,
			"gzip",
		},
		"brotliCommaStarAcceptEncodingSendsGzipNoVary": {"/mse6/gzip",
			"br,*",
			true,
			200,
			"gzip",
			false,
			"gzip",
		},
		"deflateAcceptEncodingSends406ResponseCode": {"/mse6/gzip",
			"deflate",
			true,
			406,
			"identity",
			false,
			"406",
		},
		"unknownAcceptEncodingSends406ResponseCode": {"/mse6/gzip",
			"unknown",
			true,
			406,
			"identity",
			false,
			"406",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			DownstreamContentEncodingFullIntegrity(tc.reqUrlSlug, tc.reqAcceptEncodingHeader, tc.reqSendAcceptEncodingHeader,
				tc.wantResContentEncodingHeader, tc.wantResVaryAcceptEncodingHeader, tc.wantResBodyContent, tc.wantResStatusCode, t)
		})
	}
}

func TestContentEncodingPermutationsOnProxyHandlerUpstreamBrotli(t *testing.T) {
	tests := map[string]struct {
		reqUrlSlug                      string
		reqAcceptEncodingHeader         string
		reqSendAcceptEncodingHeader     bool
		wantResStatusCode               int
		wantResContentEncodingHeader    string
		wantResVaryAcceptEncodingHeader bool
		wantResBodyContent              string
	}{
		"noAcceptEncodingSendsEncodedWithVary": {"/mse6/brotli",
			"",
			false,
			200,
			"br",
			true,
			"brotli",
		},
		"emptyAcceptEncodingSendsEncodedWithVary": {"/mse6/brotli",
			"",
			true,
			200,
			"br",
			true,
			"brotli",
		},
		"starAcceptEncodingSendsEncoded": {"/mse6/brotli",
			"*",
			true,
			200,
			"br",
			false,
			"brotli",
		},
		"starCommaGzipAcceptEncodingSendsEncoded": {"/mse6/brotli",
			"*,gzip",
			true,
			200,
			"br",
			false,
			"brotli",
		},
		"identityCommaGzipAcceptEncodingSendsEncodedWithVary": {"/mse6/brotli",
			"identity,gzip",
			true,
			200,
			"br",
			true,
			"brotli",
		},
		"gzipCommaIdentityAcceptEncodingSendsEncodedWithVary": {"/mse6/brotli",
			"gzip,identity",
			true,
			200,
			"br",
			true,
			"brotli",
		},
		"gzipCommaStarAcceptEncodingSendsEncoded": {"/mse6/brotli",
			"gzip,*",
			true,
			200,
			"br",
			false,
			"brotli",
		},
		"gzipAcceptEncodingSendsEncodedWithVary": {"/mse6/brotli",
			"gzip",
			true,
			200,
			"br",
			true,
			"brotli",
		},
		"gzipCommaUnknownAcceptEncodingSendsEncodedWithVary": {"/mse6/brotli",
			"gzip, unknown, moreunknown",
			true,
			200,
			"br",
			true,
			"brotli",
		},
		"gzipCommaBrotliAcceptEncodingSendsBrotli": {"/mse6/brotli",
			"gzip, br",
			true,
			200,
			"br",
			false,
			"brotli",
		},
		"brotliAcceptEncodingSendsBrotli": {"/mse6/brotli",
			"br",
			true,
			200,
			"br",
			false,
			"brotli",
		},
		"brotliCommaGzipAcceptEncodingSendsGzip": {"/mse6/brotli",
			"br,gzip",
			true,
			200,
			"br",
			false,
			"brotli",
		},
		"brotliCommaIdentityAcceptEncodingSendsBrotli": {"/mse6/brotli",
			"br,identity",
			true,
			200,
			"br",
			false,
			"brotli",
		},
		"brotliCommaStarAcceptEncodingSendsBrotli": {"/mse6/brotli",
			"br,*",
			true,
			200,
			"br",
			false,
			"brotli",
		},
		"deflateAcceptEncodingSends406ResponseCode": {"/mse6/brotli",
			"deflate",
			true,
			406,
			"identity",
			false,
			"406",
		},
		"unknownAcceptEncodingSends406ResponseCode": {"/mse6/brotli",
			"unknown",
			true,
			406,
			"identity",
			false,
			"406",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			DownstreamContentEncodingFullIntegrity(tc.reqUrlSlug, tc.reqAcceptEncodingHeader, tc.reqSendAcceptEncodingHeader,
				tc.wantResContentEncodingHeader, tc.wantResVaryAcceptEncodingHeader, tc.wantResBodyContent, tc.wantResStatusCode, t)
		})
	}
}

func TestContentEncodingPermutationsOnProxyHandlerUpstreamDeflate(t *testing.T) {
	tests := map[string]struct {
		reqUrlSlug                      string
		reqAcceptEncodingHeader         string
		reqSendAcceptEncodingHeader     bool
		wantResStatusCode               int
		wantResContentEncodingHeader    string
		wantResVaryAcceptEncodingHeader bool
		wantResBodyContent              string
	}{
		"noAcceptEncodingSendsEncodedWithVary": {"/mse6/deflate",
			"",
			false,
			200,
			"deflate",
			true,
			"deflate",
		},
		"emptyAcceptEncodingSendsEncodedWithVary": {"/mse6/deflate",
			"",
			true,
			200,
			"deflate",
			true,
			"deflate",
		},
		"starAcceptEncodingSendsEncoded": {"/mse6/deflate",
			"*",
			true,
			200,
			"deflate",
			false,
			"deflate",
		},
		"starCommaGzipAcceptEncodingSendsEncoded": {"/mse6/deflate",
			"*,gzip",
			true,
			200,
			"deflate",
			false,
			"deflate",
		},
		"identityCommaGzipAcceptEncodingSendsEncodedWithVary": {"/mse6/deflate",
			"identity,gzip",
			true,
			200,
			"deflate",
			true,
			"deflate",
		},
		"gzipCommaIdentityAcceptEncodingSendsEncodedWithVary": {"/mse6/deflate",
			"gzip,identity",
			true,
			200,
			"deflate",
			true,
			"deflate",
		},
		"gzipCommaStarAcceptEncodingSendsEncoded": {"/mse6/deflate",
			"gzip,*",
			true,
			200,
			"deflate",
			false,
			"deflate",
		},
		"gzipAcceptEncodingSendsEncodedWithVary": {"/mse6/deflate",
			"gzip",
			true,
			200,
			"deflate",
			true,
			"deflate",
		},
		"gzipCommaUnknownAcceptEncodingSendsEncodedWithVary": {"/mse6/deflate",
			"gzip, unknown, moreunknown",
			true,
			200,
			"deflate",
			true,
			"deflate",
		},
		"gzipCommaBrotliAcceptEncodingSendsEncodedWithVary": {"/mse6/deflate",
			"gzip, br",
			true,
			200,
			"deflate",
			true,
			"deflate",
		},
		"brotliAcceptEncodingSendsEncodedWithVary": {"/mse6/deflate",
			"br",
			true,
			200,
			"deflate",
			true,
			"deflate",
		},
		"brotliCommaGzipAcceptEncodingSendsEncodedWithVary": {"/mse6/deflate",
			"br,gzip",
			true,
			200,
			"deflate",
			true,
			"deflate",
		},
		"brotliCommaIdentityAcceptEncodingSendsEncodedWithVary": {"/mse6/deflate",
			"br,identity",
			true,
			200,
			"deflate",
			true,
			"deflate",
		},
		"brotliCommaStarAcceptEncodingSendsEncoded": {"/mse6/deflate",
			"br,*",
			true,
			200,
			"deflate",
			false,
			"deflate",
		},
		"deflateAcceptEncodingSends406ResponseCode": {"/mse6/deflate",
			"deflate",
			true,
			406,
			"identity",
			false,
			"406",
		},
		"unknownAcceptEncodingSends406ResponseCode": {"/mse6/deflate",
			"unknown",
			true,
			406,
			"identity",
			false,
			"406",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			DownstreamContentEncodingFullIntegrity(tc.reqUrlSlug, tc.reqAcceptEncodingHeader, tc.reqSendAcceptEncodingHeader,
				tc.wantResContentEncodingHeader, tc.wantResVaryAcceptEncodingHeader, tc.wantResBodyContent, tc.wantResStatusCode, t)
		})
	}
}

//no upstream attempt made.
func TestContentEncodingPermutationsOnAboutHandler(t *testing.T) {
	tests := map[string]struct {
		reqUrlSlug                      string
		reqAcceptEncodingHeader         string
		reqSendAcceptEncodingHeader     bool
		wantResStatusCode               int
		wantResContentEncodingHeader    string
		wantResVaryAcceptEncodingHeader bool
		wantResBodyContent              string
	}{
		"noAcceptEncodingSendsIdentity": {"/about",
			"",
			false,
			200,
			"identity",
			false,
			"ServerID",
		},
		"emptyAcceptEncodingSendsIdentity": {"/about",
			"",
			true,
			200,
			"identity",
			false,
			"ServerID",
		},
		"starAcceptEncodingSendsIdentity": {"/about",
			"*",
			true,
			200,
			"identity",
			false,
			"ServerID",
		},
		"starCommaGzipAcceptEncodingSendsIdentity": {"/about",
			"*,gzip",
			true,
			200,
			"identity",
			false,
			"ServerID",
		},
		"identityCommaGzipAcceptEncodingSendsIdentity": {"/about",
			"identity,gzip",
			true,
			200,
			"identity",
			false,
			"ServerID",
		},
		"gzipCommaIdentityAcceptEncodingSendsIdentity": {"/about",
			"gzip,identity",
			true,
			200,
			"identity",
			false,
			"ServerID",
		},
		"gzipCommaStarAcceptEncodingSendsIdentity": {"/about",
			"gzip,*",
			true,
			200,
			"identity",
			false,
			"ServerID",
		},
		"gzipAcceptEncodingSendsGzip": {"/about",
			"gzip",
			true,
			200,
			"gzip",
			false,
			"ServerID",
		},
		"gzipCommaUnknownAcceptEncodingSendsGzip": {"/about",
			"gzip, unknown",
			true,
			200,
			"gzip",
			false,
			"ServerID",
		},
		"gzipCommaBrotliAcceptEncodingSendsGzip": {"/about",
			"gzip, br",
			true,
			200,
			"gzip",
			false,
			"ServerID",
		},
		"brotliAcceptEncodingSendsBrotli": {"/about",
			"br",
			true,
			200,
			"br",
			false,
			"ServerID",
		},
		"brotliCommaGzipAcceptEncodingSendsGzip": {"/about",
			"br,gzip",
			true,
			200,
			"gzip",
			false,
			"ServerID",
		},
		"deflateAcceptEncodingSends406ResponseCode": {"/about",
			"deflate",
			true,
			406,
			"identity",
			false,
			"406",
		},
		"unknownAcceptEncodingSends406ResponseCode": {"/about",
			"unknown",
			true,
			406,
			"identity",
			false,
			"406",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			DownstreamContentEncodingFullIntegrity(tc.reqUrlSlug, tc.reqAcceptEncodingHeader, tc.reqSendAcceptEncodingHeader,
				tc.wantResContentEncodingHeader, tc.wantResVaryAcceptEncodingHeader, tc.wantResBodyContent, tc.wantResStatusCode, t)
		})
	}
}

//no upstream attempt made
func TestContentEncodingPermutationsOnStatusCodeResponseInProxyHandler(t *testing.T) {
	tests := map[string]struct {
		reqUrlSlug                      string
		reqAcceptEncodingHeader         string
		reqSendAcceptEncodingHeader     bool
		wantResStatusCode               int
		wantResContentEncodingHeader    string
		wantResVaryAcceptEncodingHeader bool
		wantResBodyContent              string
	}{
		"noAcceptEncodingSendsIdentity": {"/badslug",
			"",
			false,
			404,
			"identity",
			false,
			"404",
		},
		"emptyAcceptEncodingSendsIdentity": {"/badslug",
			"",
			true,
			404,
			"identity",
			false,
			"404",
		},
		"starAcceptEncodingSendsIdentity": {"/badslug",
			"*",
			true,
			404,
			"identity",
			false,
			"404",
		},
		"starCommaGzipAcceptEncodingSendsIdentity": {"/badslug",
			"*,gzip",
			true,
			404,
			"identity",
			false,
			"404",
		},
		"identityCommaGzipAcceptEncodingSendsIdentity": {"/badslug",
			"identity,gzip",
			true,
			404,
			"identity",
			false,
			"404",
		},
		"gzipCommaIdentityAcceptEncodingSendsIdentity": {"/badslug",
			"gzip,identity",
			true,
			404,
			"identity",
			false,
			"404",
		},
		"gzipCommaStarAcceptEncodingSendsIdentity": {"/badslug",
			"gzip,*",
			true,
			404,
			"identity",
			false,
			"404",
		},
		"gzipAcceptEncodingSendsGzip": {"/badslug",
			"gzip",
			true,
			404,
			"gzip",
			false,
			"404",
		},
		"gzipCommaUnknownAcceptEncodingSendsGzip": {"/badslug",
			"gzip, unknown",
			true,
			404,
			"gzip",
			false,
			"404",
		},
		"gzipCommaBrotliAcceptEncodingSendsGzip": {"/badslug",
			"gzip, br",
			true,
			404,
			"gzip",
			false,
			"404",
		},
		"brotliAcceptEncodingSendsBrotli": {"/badslug",
			"br",
			true,
			404,
			"br",
			false,
			"404",
		},
		"brotliCommaGzipAcceptEncodingSendsGzip": {"/badslug",
			"br,gzip",
			true,
			404,
			"gzip",
			false,
			"404",
		},
		"deflateAcceptEncodingSends406ResponseCode": {"/badslug",
			"deflate",
			true,
			406,
			"identity",
			false,
			"406",
		},
		"unknownAcceptEncodingSends406ResponseCode": {"/badslug",
			"unknown",
			true,
			406,
			"identity",
			false,
			"406",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			DownstreamContentEncodingFullIntegrity(tc.reqUrlSlug, tc.reqAcceptEncodingHeader, tc.reqSendAcceptEncodingHeader,
				tc.wantResContentEncodingHeader, tc.wantResVaryAcceptEncodingHeader, tc.wantResBodyContent, tc.wantResStatusCode, t)
		})
	}
}

func DownstreamContentEncodingFullIntegrity(reqUrlSlug string, reqAcceptEncodingHeader string, reqSendAcceptEncodingHeader bool,
	wantResContentEncodingHeader string, wantResVaryAcceptEncodingHeader bool, wantResBodyContent string,
	wantResStatusCode int, t *testing.T) []byte {

	client := &http.Client{
		Transport: &http.Transport{DisableCompression: true},
	}
	req, _ := http.NewRequest("GET", "http://localhost:8080"+reqUrlSlug, nil)
	if reqSendAcceptEncodingHeader {
		req.Header.Add(j8a.AcceptEncodingS, reqAcceptEncodingHeader)
	} else {
		req.Header.Del(j8a.AcceptEncodingS)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("error connecting to server, cause: %s", err)
	}

	gotStatusCode := resp.StatusCode
	if gotStatusCode != wantResStatusCode {
		t.Errorf("want status code %v, but got %v instead", wantResStatusCode, gotStatusCode)
	}

	gotce := resp.Header.Get("Content-Encoding")
	if gotce != wantResContentEncodingHeader {
		t.Errorf("want content encoding %s, but got %s instead", wantResContentEncodingHeader, gotce)
	}

	gotVary := resp.Header.Get("Vary")
	if wantResVaryAcceptEncodingHeader == true {
		if gotVary != "Accept-Encoding" {
			t.Errorf("want Vary: Accept-Encoding, but got %s instead", gotVary)
		}
	} else {
		if len(gotVary) > 0 {
			t.Errorf("no vary header should be sent, but got %s", gotVary)
		}
	}

	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	body, _ := ioutil.ReadAll(resp.Body)
	if len(wantResBodyContent) > 0 {
		if wantResContentEncodingHeader == "br" {
			body = *j8a.BrotliDecode(body)
		} else if wantResContentEncodingHeader == "gzip" {
			body = *j8a.Gunzip(body)
		} else if wantResContentEncodingHeader == "deflate" {
			body, _ = ioutil.ReadAll(flate.NewReader(bytes.NewBuffer(body)))
		}

		if !strings.Contains(string(body), wantResBodyContent) {
			t.Errorf("want body response %v, but got (decoded) %v", wantResBodyContent, string(body))
		}
	}
	return body
}
