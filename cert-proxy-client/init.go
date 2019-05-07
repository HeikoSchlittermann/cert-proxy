package main

import (
	"cert-proxy/cert-proxy-client/cert"
	"cert-proxy/cert-proxy-client/secret"
	"cert-proxy/internal/program"
	. "cert-proxy/internal/shared"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
)

func init() {

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [<CN>]...\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprint(os.Stderr, `
The cert-proxy-client exits with 0 on success, and with any other value if there
is a problem.

¹) The hook script gets called as
       <script> deploy_cert <DOMAIN> <KEYFILE> <CERTFILE> <CHAINFILE> <TIMESTAMP>
    or
       <script> deploy_cert <DOMAIN> <BUNDLEFILE> <TIMESTAMP>
    Additionally the corresponding environment variables are set (possibly overriding
    existing environment variables with the same name.

²) The password for protecting the P12 file may be given in one of the following notations:
    - pass:<password>
    - file:<file containing the password>
    - env:<environment variable containing the password>
`)
	}

	// On Windows: outFormat is PKCS12 and Item is "bundle"
	// On Linux: outFormat is PEM and Items are "cert", "chain", "fullchain", "privkey"
	// -outformat PEM    implies cert, chain, fullchain, privkey
	// -outformat PKCS12 implies bundle

	var printVersion bool
	var logOutput out = STDERR

	flag.BoolVar(&cert.UseSymlink, "symlink", cert.UseSymlink, "Use symlinks for current files")
	flag.BoolVar(&opt.Auto, "auto", true, "Auto mode (fetch all CNs the server provides us)")
	flag.BoolVar(&opt.Verbose, "verbose", false, "Verbose output")
	flag.BoolVar(&printVersion, "version", false, "current version ("+program.Version+")")
	flag.BoolVar(&cert.Force, "force", false, "Force download, even if not modified")
	flag.IntVar(&opt.Jobs, "jobs", runtime.NumCPU(), "Maximum number of parallel running jobs")
	flag.StringVar(&opt.Certbase, "certbase", "certs", "Base dir for downloaded certs")
	flag.StringVar(&opt.CNfile, "cnfile", "", "CN list file (use - for stdin)")
	flag.StringVar(&opt.Connect, "connect", "https://localhost:4433", "Address of cert proxy server")
	flag.StringVar(&opt.Hook, "hook", "", "Hook script¹")
	flag.StringVar(&opt.Passout, "passout", "", "Passwort to protect the PKCS12²")
	flag.StringVar(&opt.ServerCN, "servername", "cert-proxy", "Name (CN) of the cert proxy certificate")
	flag.StringVar(&opt.SSLFile, "sslfile", "ssl.pem", "SSL auth file (crt+key+ca) PEM")
	flag.Var(&logOutput, "stderr", "Redirect stderr (stderr|stdout)")
	flag.Var(&opt.Format, "format", "Format of the requested certificate(s) (PEM|PKCS12)")
	flag.Parse()

	if logOutput == `STDOUT` {
		*os.Stderr = *os.Stdout
	}

	if printVersion {
		fmt.Println(program.Version, program.Name, program.Path)
		os.Exit(0)
	}

	if !opt.Auto && opt.CNfile == "" && flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	for _, v := range flag.Args() {
		CNs.Add(v)
	}

	if opt.Verbose {
		Verbose = log.New(os.Stderr, ``, log.Flags()).Printf
	}

	if opt.Passout != "" {
		var err error
		opt.Passout, err = secret.Read(opt.Passout)
		if err != nil {
			log.Fatal(err)
		}
	}
}
