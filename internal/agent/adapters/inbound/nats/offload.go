package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/shared/ports/outbound"
)

// S3Reference represents the structured JSON offloading payload reference.
type S3Reference struct {
	Type        string `json:"_type"`
	Bucket      string `json:"bucket"`
	Key         string `json:"key"`
	SizeBytes   int64  `json:"size_bytes"`
	ContentType string `json:"content_type"`
}

// NormalizeBucketName normalizes a tenant name to meet S3 bucket requirements.
func NormalizeBucketName(tenantName string) string {
	name := strings.ToLower(tenantName)
	var sb strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			sb.WriteRune(r)
		} else {
			sb.WriteRune('-')
		}
	}
	name = sb.String()
	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}
	name = strings.Trim(name, "-")
	if len(name) > 63 {
		name = name[:63]
		name = strings.TrimSuffix(name, "-")
	}
	return name
}

// unescapeReader decodes JSON string escapes streamingly to avoid flat in-memory slice allocations.
type unescapeReader struct {
	data []byte
	pos  int
}

func (r *unescapeReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = 0
	for r.pos < len(r.data) && n < len(p) {
		c := r.data[r.pos]
		if c == '\\' {
			if r.pos+1 >= len(r.data) {
				p[n] = c
				n++
				r.pos++
				continue
			}
			next := r.data[r.pos+1]
			switch next {
			case 'n':
				p[n] = '\n'
				r.pos += 2
			case 't':
				p[n] = '\t'
				r.pos += 2
			case 'r':
				p[n] = '\r'
				r.pos += 2
			case '\\':
				p[n] = '\\'
				r.pos += 2
			case '"':
				p[n] = '"'
				r.pos += 2
			case '/':
				p[n] = '/'
				r.pos += 2
			case 'u':
				if r.pos+5 < len(r.data) {
					hexVal := 0
					ok := true
					for i := 0; i < 4; i++ {
						h := r.data[r.pos+2+i]
						hexVal <<= 4
						if h >= '0' && h <= '9' {
							hexVal += int(h - '0')
						} else if h >= 'a' && h <= 'f' {
							hexVal += int(h - 'a' + 10)
						} else if h >= 'A' && h <= 'F' {
							hexVal += int(h - 'A' + 10)
						} else {
							ok = false
							break
						}
					}
					if ok {
						p[n] = byte(hexVal)
						r.pos += 6
					} else {
						p[n] = c
						r.pos++
					}
				} else {
					p[n] = c
					r.pos++
				}
			default:
				p[n] = c
				r.pos++
			}
		} else {
			p[n] = c
			r.pos++
		}
		n++
	}
	return n, nil
}

// OffloadPayload uploads a payload to S3 and returns the serialized S3Reference.
func OffloadPayload(ctx context.Context, blobStore outbound.BlobStore, communityID, agentName, threadID, tenantID string, payloadBytes []byte) (string, error) {
	bucketName := NormalizeBucketName(tenantID)
	if bucketName == "" {
		bucketName = "default"
	}

	objectID := uuid.New().String()
	s3Key := fmt.Sprintf("%s/ingress/%s/%s/%s", communityID, agentName, threadID, objectID)

	reader := &unescapeReader{data: payloadBytes}
	_, err := blobStore.Put(ctx, s3Key, reader, "text/plain")
	if err != nil {
		return "", fmt.Errorf("blob store put: %w", err)
	}

	ref := S3Reference{
		Type:        "s3_reference",
		Bucket:      bucketName,
		Key:         s3Key,
		SizeBytes:   int64(len(payloadBytes)),
		ContentType: "text/plain",
	}

	refBytes, err := json.Marshal(ref)
	if err != nil {
		return "", fmt.Errorf("marshal S3 reference: %w", err)
	}

	return string(refBytes), nil
}
