package ignite3

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/GHKyle/ignite3-go-client/binary/errors"
)

// ResponseHandshake is struct handshake response. It is the only response to have MAGIC_BYTES header
type ResponseHandshake struct {
	// Success flag
	Success bool
	// Server version major, minor, patch
	Major, Minor, Patch int
	ErrorUuid          Uuid
	ErrorMessage       string
	ErrorStackTrace    string
	ErrorDetails       string

	ClientType         int8
	IdleTimeout        int64
	ClusterNodeId      Uuid
	ClusterNodeName    string
	ClusterIdsCount    uint8
	ClusterIds         []Uuid
	ClusterName        string
	ObservableTS       int64
	DBMSVerMajor       uint8
	DBMSVerMinor       uint8
	DBMSVerMaintenance uint8
	DBMSVerPatch       string
	DBMSVerPreRelease  string
	Features           []byte	// FIXME 

	response
}

/*
 * Exemple with cluster named "test000" and node "defaultNode"
0000   00 00 00 00 00 00 00 00 00 00 00 00 86 dd 60 05
0010   22 ee 00 7a 06 40 00 00 00 00 00 00 00 00 00 00
0020   00 00 00 00 00 01 00 00 00 00 00 00 00 00 00 00
0030   00 00 00 00 00 01 2a 30 93 1e 20 10 6a d2 a4 67
0040   18 a0 80 18 02 00 00 82 00 00 01 01 08 0a d9 1f
0050   14 23 d9 1f 14 22 49 47 4e 49 00 00 00 52 03 00
0060   00 c0 00 d8 03 4c 47 8a ae a3 70 5c da fb 05 40
0070   d4 9b 83 d7 81 d9 0b 64 65 66 61 75 6c 74 4e 6f
0080   64 65 01 d8 03 d7 49 0c 12 50 99 f2 b9 f7 d4 52
0090   31 5f e8 d4 90 a7 74 65 73 74 30 30 30 cf 01 9a
00a0   d0 d6 a7 91 00 00 03 01 00 c0 c0 c4 02 2a 0a 00
Data begin at 0x56 :
49 47 4e 49        // 1. Magic bytes "IGNI"
00 00 00 52        // 2. Payload size : 82 bytes. Alll the following is 3. Payload

03 00 00           // Protocol version : 3.0.0
c0                 // MSGPACK_OBJECT_NIL = No error here
00                 // IdleTimeoutMS

d8 03 4c 47 8a ae a3 70 5c da fb 05 40 d4 9b 83 d7 81 // UUID v4.9
d9 0b 
64 65 66 61 75 6c 74 4e 6f 64 65 // Cluster node name: "defaultNode"
01                               // cluster_ids_len ?
d8 03 d7 49 0c 12 50 99 f2 b9 f7 d4 52 31 5f e8 d4 90 
a7 74 65 73 74 30 30 30          // cluster_name : "test000"
cf 01 9a d0 d6 a7 91 00 00 03 01 00 c0 c0 c4 02 2a 0a 00

*/

/* Exemple d'un handshake en erreur (mauvaise version de protocole présenté par le client)
 * 
49 47 4e 49 
00 00 00 bc 
03 00 00 
d8 03 fa 46 62 48 ed 11 02 89 f3 9d cc ed 78 6e 28 9a  // UUID v4.9 : 890211ed-4862-46fa-9a28-6e78edcc9d
ce 00 03 00 03 // MsgPack int32 : 00030003
d9 26 6f 72 67 2e 61 70 61 63 68 65 2e 69 67 6e 69 74 65 2e 6c 61 6e 67 2e 49 67 6e 69 74 65 45 78 63 65 70 74 69 6f 6e // string de 0x26 = 38d longueur : "org.apache.ignite.lang.IgniteException"
d9 1a 55 6e 73 75 70 70 6f 72 74 65 64 20 76 65 72 73 69 6f 6e 3a 20 31 2e 30 2e 30 // string de 0x1a = 26 car : "Unsupported version: 1.0.0"
da 00 5a 54 6f 20 73 65 65 20 74 68 65 20 66 75 6c 6c 20 73 74 61 63 6b 20 74 72 61 63 65 20 73 65 74 20 63 6c 69 65 6e 74 43 6f 6e 6e 65 63 74 6f 72 2e 73 65 6e 64 53 65 72 76 65 72 45 78 63 65 70 74 69 6f 6e 53 74 61 63 6b 54 72 61 63 65 54 6f 43 6c 69 65 6e 74 3a 74 72 75 65 // string utf16 de 0x5a = 90d caracteres : "To see the full stack trace set clientConnector.sendServerExceptionStackTraceToClient:true"
c0 // MsgPack Nil
 * 
 */
 

// v3_OK
// ReadFrom is function to read request data from io.Reader.
// Returns read bytes.
func (r *ResponseHandshake) ReadFrom(rr *bufio.Reader) (int64, error) {
	r.Success = false
	// Read magic bytes
	magic, err := ReadRawByteArray(rr, 4)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to read response magic bytes")
	}
	if !strings.EqualFold(string(magic), string(MAGIC_BYTES)) {
		return 0, fmt.Errorf("response magic bytes is not %s", string(MAGIC_BYTES))
	}

	l, err := r.response.ReadFrom(rr)
	if err != nil {
		return l, err
	}

	v, err := ReadPackedInt8(r.response.message)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to read server version major")
	}
	r.Major = int(v)

	v, err = ReadPackedInt8(r.response.message)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to read server version minor")
	}
	r.Minor = int(v)

	v, err = ReadPackedInt8(r.response.message)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to read server version patch")
	}
	r.Patch = int(v)
	
	if Debug { fmt.Printf("Server version support : v%d.%d.%d\n", r.Major, r.Minor, r.Patch) }

	// Either a "Nil" msgpack, or a traceId (Uuid) followed by error message (res.error = try_read_error(reader); dans messages.cpp)
	interf, err := TryReadError(r.response.message)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to read error message")
	} else {
		switch value := interf.(type) {
			case nil:
				//fmt.Printf("DEBUG: TryReadError returned \"no error\"\n")
				r.Success = true
			// If an uuid, it's an error traceId, lets continue reading error code as int, then error message as string
			case *Uuid:
				//fmt.Printf("DEBUG: TryReadError got a Uuid result : %s\n", value)
				r.ErrorUuid = *value
				// Read IDK, maybe a type? (following this value there is 3 string, this value is 0x30003)
				var idk int64
				if idk, err = ReadPackedInt64(r.response.message); err != nil {
					fmt.Printf("ERROR: failed to read error code : %v\n", err)
					return 0, fmt.Errorf("failed to read error code : %v\n", err)
				}
				fmt.Printf("DEBUG: Got what, an error code? Size of following data? Value = %x\n", idk)
				if r.ErrorMessage, err = ReadPackedString(r.response.message); err != nil {
					fmt.Printf("ERROR: failed to read error message : %v\n", err)
					return 0, fmt.Errorf("failed to read error message : %v\n", err)
				}
				if r.ErrorStackTrace, err = ReadPackedString(r.response.message); err != nil {
					fmt.Printf("ERROR: failed to read error stack trace : %v\n", err)
					return 0, fmt.Errorf("failed to read error stack trace : %v\n", err)
				}
				if r.ErrorDetails, err = ReadPackedString(r.response.message); err != nil {
					fmt.Printf("ERROR: failed to read error details : %v\n", err)
					return 0, fmt.Errorf("failed to read error details : %v\n", err)
				}
				//r.ErrorMessage = "TO_BE_DONE"
				//return 0, errors.Wrapf(err, "handshake error %s : %s", r.ErrorUuid, r.ErrorMessage)
				return 0, fmt.Errorf("handshake error %s : %s %s %s", r.ErrorUuid, r.ErrorMessage, r.ErrorStackTrace, r.ErrorDetails)
			default:
				fmt.Printf("ERROR: TryReadError got an unknown type : %T\n", value)
		}
	}
	
	//r.Message

	// FIXME: Make sure idle timeout is always 0, and/or always 1 byte
	t, err := ReadPackedInt8(r.response.message)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to read idle timeout")
	}
	r.IdleTimeout = int64(t)
	
	//fmt.Printf("r.IdleTimeout : %d\n", r.IdleTimeout)
	
	r.ClusterNodeId, err = ReadPackedUUID(r.response.message)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to read cluster node ID")
	}
	
	// REMOVE ME
	if Debug { fmt.Printf("Cluster Node ID : %s\n", r.ClusterNodeId) }
	
	r.ClusterNodeName, err = ReadPackedString(r.response.message)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to read cluster node name")
	}
	// REMOVE ME
	if Debug { fmt.Printf("Cluster Node name : %s\n", r.ClusterNodeName) }
	
	r.ClusterIdsCount, err = ReadPackedUint8(r.response.message)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to read cluster id count")
	}
	if Debug { fmt.Printf("Cluster ID count : %d\n", r.ClusterIdsCount) }

	var cu Uuid
	for i := 0 ; i < int(r.ClusterIdsCount) ; i++ {
		cu, err = ReadPackedUUID(r.response.message)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to read cluster node ID %d", i)
		}
		r.ClusterIds = append(r.ClusterIds, cu)
	}
	if Debug { for _, c := range r.ClusterIds {
		fmt.Printf("Cluster ID : %s\n", c)
	}}

	r.ClusterName, err = ReadPackedString(r.response.message)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to read cluster name")
	}
	
	if Debug { fmt.Printf("Cluster name : %s\n", r.ClusterName) }
	
	r.ObservableTS, err = ReadPackedInt64(r.response.message)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to read observable timestamp")
	}
	
	//fmt.Printf("Observable timestamp : %d\n", r.ObservableTS)
	
	r.DBMSVerMajor, err = ReadPackedUint8(r.response.message)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to read DBMS major version")
	}
	if Debug { fmt.Printf("DBMS major version : %d\n", r.DBMSVerMajor) }
	
	r.DBMSVerMinor, err = ReadPackedUint8(r.response.message)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to read DBMS minor version")
	}
	if Debug { fmt.Printf("DBMS minor version : %d\n", r.DBMSVerMinor) }
	
	r.DBMSVerMaintenance, err = ReadPackedUint8(r.response.message)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to read DBMS maintenance version")
	}
	if Debug { fmt.Printf("DBMS maintenance version : %d\n", r.DBMSVerMaintenance) }
	
	r.DBMSVerPatch, err = ReadPackedString(r.response.message)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to read DBMS patch version")
	}
	if Debug { fmt.Printf("DBMS patch version : %s\n", r.DBMSVerPatch) }
	
	r.DBMSVerPreRelease, err = ReadPackedString(r.response.message)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to read DBMS pre release version")
	}
	if Debug { fmt.Printf("DBMS pre release version : %s\n", r.DBMSVerPreRelease) }
	
	// TODO : decode Features
	r.Features, err = ReadPackedBytes(r.response.message)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to read features")
	}
	//fmt.Printf("features : %x = %b\n", r.Features, r.Features)
	r.Success = true

	// Return length read, excluding MAGIC_BYTES and length of "payload length"
	return l - 4 , nil
}
