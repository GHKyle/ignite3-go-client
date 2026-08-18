package ignite3

import (
	"io"
	"bytes"

	"github.com/GHKyle/ignite3-go-client/binary/errors"
)

const (
	AuthInfoUsernameLabel = "authn-identity"
	AuthInfoPasswordLabel = "authn-secret"
	AuthInfoType          = "authn-type"
)

// RequestHandshake is struct handshake request
type RequestHandshake struct {
	major, minor, patch int
	authInfo     map[string]string

	request
}

// WriteTo is function to write handshake request data to io.Writer.
// Returns written bytes.
func (r *RequestHandshake) WriteTo(w io.Writer) (int64, error) {
	var buffer bytes.Buffer

	// FIXME : see modules/platform/cpp/ignite/protocol/messages.cpp
	features := []byte{0x04}

	// Java use short (int16), but on the wire they are bytes (int8) (see ProtocolVersion.java)
	if err := WritePackedByte(&buffer, byte(r.major)); err != nil {
		return 0, errors.Wrapf(err, "failed to write handshake version major")
	}
	if err := WritePackedByte(&buffer, byte(r.minor)); err != nil {
		return 0, errors.Wrapf(err, "failed to write handshake version minor")
	}
	if err := WritePackedByte(&buffer, byte(r.patch)); err != nil {
		return 0, errors.Wrapf(err, "failed to write handshake version patch")
	}
	// CLIENT_TYPE_GENERAL in HandshakeUtils.java
	if err := WritePackedByte(&buffer, 2); err != nil {
		return 0, errors.Wrapf(err, "failed to write handshake client type code")
	}
	//if err := WritePackedBinary(&buffer, features); err != nil {
	if err := WritePackedBytes(&buffer, features); err != nil {
		return 0, errors.Wrapf(err, "failed to write handshake supported bitmask features code")
	}
	if err := WritePackedMap(&buffer, r.authInfo); err != nil {
		return 0, errors.Wrapf(err, "failed to write handshake msgpacked authentication")
	}
	
	// Now we can write to stream
	if err := WriteRawBytes(r, MAGIC_BYTES); err != nil {
		return 0, errors.Wrapf(err, "failed to write handshake magic bytes")
	}
	l := int32(buffer.Len())
	if err := WriteRawInt32(r, l); err != nil {
		return 0, errors.Wrapf(err, "failed to write handshake request length")
	}
	if err := WriteRawBytes(r, buffer.Bytes()); err != nil {
		return 0, errors.Wrapf(err, "failed to write handshake payload")
	}

	// write request
	n, err := r.request.WriteTo(w)
	return n, err
}

// NewRequestHandshake creates new handshake request object
// Only basic authentication supported
func NewRequestHandshake(major, minor, patch int, username, password string) *RequestHandshake {
	r := RequestHandshake{request: newRequest(),
		major: major, minor: minor, patch: patch}
	r.authInfo = make(map[string]string)
	r.authInfo[AuthInfoUsernameLabel] = username
	r.authInfo[AuthInfoPasswordLabel] = password
	r.authInfo[AuthInfoType] = "basic"
	return &r
}
