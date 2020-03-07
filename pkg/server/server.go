package server

import (
	"fmt"
	"io"
	"net/http"

	"github.com/AlexisDuf/k8sWebhooks/pkg/admissioncontroller"
	"github.com/AlexisDuf/k8sWebhooks/pkg/config"
	"github.com/gorilla/mux"
	"github.com/spf13/cobra"
	"k8s.io/klog"
)

var (
	certFile     string
	keyFile      string
	port         int
	sidecarImage string
)

type webhookServer struct {
	server *http.Server
	config *config.Sidecar
	router *mux.Router
}

// CmdServer is used for webhooks.
var CmdServer = &cobra.Command{
	Use:   "server",
	Short: "Starts a HTTP server, useful for testing MutatingAdmissionWebhook and ValidatingAdmissionWebhook",
	Long: `Starts a HTTP server, useful for testing MutatingAdmissionWebhook and ValidatingAdmissionWebhook.
After deploying it to Kubernetes cluster, the Administrator needs to create a ValidatingWebhookConfiguration
in the Kubernetes cluster to register remote webhook admission controllers.`,
	Args: cobra.MaximumNArgs(0),
	Run:  main,
}

func init() {
	CmdServer.Flags().StringVar(&certFile, "tls-cert-file", "/Users/alexisdufour/Documents/poc/k8sWebhooks/certs/server.csr",
		"File containing the default x509 Certificate for HTTPS. (CA cert, if any, concatenated after server cert).")
	CmdServer.Flags().StringVar(&keyFile, "tls-private-key-file", "/Users/alexisdufour/Documents/poc/k8sWebhooks/certs/server-key.pem",
		"File containing the default x509 private key matching --tls-cert-file.")
	CmdServer.Flags().IntVar(&port, "port", 443,
		"Secure port that the webhook listens on")
	CmdServer.Flags().StringVar(&sidecarImage, "sidecar-image", "",
		"Image to be used as the injected sidecar")
}

// An handler to say we are alive
func (s *webhookServer) handleHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"alive": true}`)
	}
}

func (s *webhookServer) handleSidecar() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admissioncontroller.ServeMutatePods(w, r)
	}
}

func (s *webhookServer) routes() {
	s.router.HandleFunc("/healthz", s.handleHealth()).Methods("GET")
	s.router.HandleFunc("/sidecar", s.handleSidecar()).Methods("POST")
}

func main(cmd *cobra.Command, args []string) {
	// config := Config{
	// 	CertFile: certFile,
	// 	KeyFile:  keyFile,
	// }

	var cfg config.Sidecar
	err := config.LoadFromYAML(sidecarImage, &cfg)

	if err != nil {
		klog.Fatal("Cannot read config from file %s", sidecarImage)
	}

	r := mux.NewRouter()

	server := &http.Server{
		Addr: fmt.Sprintf(":%d", port),
		// TLSConfig: configTLS(config)
	}

	webhookServer := webhookServer{server, &cfg, r}
	webhookServer.routes()
	webhookServer.server.Handler = r

	// klog.Fatal(webhookServer.server.ListenAndServe("", ""))
	klog.Fatal(webhookServer.server.ListenAndServe())

}
