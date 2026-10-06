// Package consumer exposes the host identity stored in the system consumer certificate.
//
// Get reads the certificate at /etc/pki/consumer/cert.pem by default. Set
// RHC_HEARTBEAT_CLIENT_CERT_PATH to use a different certificate. The returned Identity contains
// the system UUID and organization ID extracted from the certificate subject.
//
//	identity, err := consumer.Get()
//	if err != nil {
//		return err
//	}
//	_ = identity.UUID
package consumer
