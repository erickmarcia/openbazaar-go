package cmd

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"gx/ipfs/QmPpYHPRGVpSJTkQDQDwTYZ1cYUR2NM4HS6M3iAXi8aoUa/go-libp2p-kad-dht"
	"gx/ipfs/QmPpYHPRGVpSJTkQDQDwTYZ1cYUR2NM4HS6M3iAXi8aoUa/go-libp2p-kad-dht/opts"
	ma "gx/ipfs/QmT4U94DnD8FRfqr21obWY32HLM5VExccPKMjQHofeYqr9/go-multiaddr"
	"gx/ipfs/QmTRhk7cgjUf2gfQ3p2M9KPECNZEW9XUrmHcFCgog4cPgB/go-libp2p-peer"
	pstore "gx/ipfs/QmTTJcDL3gsnGDALjh2fDGg1onGRUdVgNL2hU2WEZcVrMX/go-libp2p-peerstore"
	"gx/ipfs/QmUDTcnDp2WssbmiDLC6aYurUeyt7QeRakHUQMxA2mZ5iB/go-libp2p"
	oniontp "gx/ipfs/QmXsGirmFALkAbRuj2yi991xamiqHBiU4wCmXv2mNsnFUq/go-onion-transport"
	ipfslogging "gx/ipfs/QmZChCsSt8DctjceaL56Eibc29CVQq4dGKRXC5JRZ6Ppae/go-log/writer"
	"gx/ipfs/Qma9Eqp16mNHDX1EL73pcxhFfzbyXVcAYtaDd1xdmDRDtL/go-libp2p-record"
	ipnspb "gx/ipfs/QmaRFtZhVAwXBk4Z3zEsvjScH9fjsDZmhXfa1Gm8eMb9cg/go-ipns/pb"
	ds "gx/ipfs/QmaRb5yNXKonhbkpNxNawoydk4N6es6b4fPj19sjEKsh5D/go-datastore"
	"gx/ipfs/Qmaabb1tJZ2CX5cp6MuuiGgns71NYoxdgQP6Xdid1dVceC/go-multiaddr-net"
	"gx/ipfs/QmcQ81jSyWCp1jpkQ8CMbtpXT3jK7Wg6ZtYmoyWFgBoF9c/go-libp2p-routing"
	p2phost "gx/ipfs/QmdJfsSbKSZnMkfZ1kpopiyB9i3Hd6cp8VKWZmtWPa7Moc/go-libp2p-host"
	"gx/ipfs/QmdxUuburamoF6zF9qjeQC4WYcWGbWuRmdLacMEsW8ioD8/gogo-protobuf/proto"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/OpenBazaar/openbazaar-go/api"
	"github.com/OpenBazaar/openbazaar-go/core"
	"github.com/OpenBazaar/openbazaar-go/ipfs"
	obnet "github.com/OpenBazaar/openbazaar-go/net"
	"github.com/OpenBazaar/openbazaar-go/net/service"
	"github.com/OpenBazaar/openbazaar-go/repo"
	"github.com/OpenBazaar/openbazaar-go/repo/db"
	"github.com/OpenBazaar/openbazaar-go/repo/migrations"
	"github.com/OpenBazaar/openbazaar-go/schema"
	sto "github.com/OpenBazaar/openbazaar-go/storage"
	"github.com/OpenBazaar/openbazaar-go/storage/dropbox"
	"github.com/OpenBazaar/openbazaar-go/storage/selfhosted"
	"github.com/OpenBazaar/openbazaar-go/wallet"
	lis "github.com/OpenBazaar/openbazaar-go/wallet/listeners"
	"github.com/OpenBazaar/openbazaar-go/wallet/resync"
	wi "github.com/OpenBazaar/wallet-interface"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil/base58"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/fatih/color"
	"github.com/ipfs/go-ipfs/commands"
	ipfscore "github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/corehttp"
	"github.com/ipfs/go-ipfs/namesys"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	"github.com/natefinch/lumberjack"
	"github.com/op/go-logging"
	"github.com/tyler-smith/go-bip39"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/net/proxy"
	bitswap "gx/ipfs/QmNkxFCmPtr2RQxjZNRCNryLud4L9wMEiBJsLgF14MqTHj/go-bitswap/network"
	"gx/ipfs/QmPEpj17FDRpc7K1aArKZp3RsHtzRMKykeK9GVgn4WQGPR/go-ipfs-config"
)

var stdoutLogFormat = logging.MustStringFormatter(
	`%{color:reset}%{color}%{time:15:04:05.000} [%{level}] [%{module}/%{shortfunc}] %{message}`,
)

var fileLogFormat = logging.MustStringFormatter(
	`%{time:15:04:05.000} [%{level}] [%{module}/%{shortfunc}] %{message}`,
)

var (
	ErrNoGateways = errors.New("No gateway addresses configured")
)

type Start struct {
	Password             string   `short:"p" long:"password" description:"the encryption password if the database is encrypted"`
	Testnet              bool     `short:"t" long:"testnet" description:"use the test network"`
	Regtest              bool     `short:"r" long:"regtest" description:"run in regression test mode"`
	LogLevel             string   `short:"l" long:"loglevel" description:"set the logging level [debug, info, notice, warning, error, critical]" default:"debug"`
	NoLogFiles           bool     `short:"f" long:"nologfiles" description:"save logs on disk"`
	AllowIP              []string `short:"a" long:"allowip" description:"only allow API connections from these IPs"`
	STUN                 bool     `short:"s" long:"stun" description:"use stun on µTP IPv4"`
	DataDir              string   `short:"d" long:"datadir" description:"specify the data directory to be used"`
	AuthCookie           string   `short:"c" long:"authcookie" description:"turn on API authentication and use this specific cookie"`
	UserAgent            string   `short:"u" long:"useragent" description:"add a custom user-agent field"`
	Verbose              bool     `short:"v" long:"verbose" description:"print openbazaar logs to stdout"`
	TorPassword          string   `long:"torpassword" description:"Set the tor control password. This will override the tor password in the config."`
	Tor                  bool     `long:"tor" description:"Automatically configure the daemon to run as a Tor hidden service and use Tor exclusively. Requires Tor to be running."`
	DualStack            bool     `long:"dualstack" description:"Automatically configure the daemon to run as a Tor hidden service IN ADDITION to using the clear internet. Requires Tor to be running. WARNING: this mode is not private"`
	DisableWallet        bool     `long:"disablewallet" description:"disable the wallet functionality of the node"`
	DisableExchangeRates bool     `long:"disableexchangerates" description:"disable the exchange rate service to prevent api queries"`
	Storage              string   `long:"storage" description:"set the outgoing message storage option [self-hosted, dropbox] default=self-hosted"`
	BitcoinCash          bool     `long:"bitcoincash" description:"use a Bitcoin Cash wallet in a dedicated data directory"`
	ZCash                string   `long:"zcash" description:"use a ZCash wallet in a dedicated data directory. To use this you must pass in the location of the zcashd binary."`
}

func (x *Start) Execute(args []string) error {
	printSplashScreen(x.Verbose)

	if x.Testnet && x.Regtest {
		return errors.New("Invalid combination of testnet and regtest modes")
	}

	if x.Tor && x.DualStack {
		return errors.New("Invalid combination of tor and dual stack modes")
	}

	isTestnet := false
	if x.Testnet || x.Regtest {
		isTestnet = true
	}
	if x.BitcoinCash && x.ZCash != "" {
		return errors.New("Bitcoin Cash and ZCash cannot be used at the same time")
	}

	// Set repo path
	repoPath, err := repo.GetRepoPath(isTestnet)
	if err != nil {
		return err
	}
	if x.BitcoinCash {
		repoPath += "-bitcoincash"
	} else if x.ZCash != "" {
		repoPath += "-zcash"
	}
	if x.DataDir != "" {
		repoPath = x.DataDir
	}

	repoLockFile := filepath.Join(repoPath, fsrepo.LockFile)
	os.Remove(repoLockFile)

	// Logging
	w := &lumberjack.Logger{
		Filename:   path.Join(repoPath, "logs", "ob.log"),
		MaxSize:    10, // Megabytes
		MaxBackups: 3,
		MaxAge:     30, // Days
	}
	var backendStdoutFormatter logging.Backend
	if x.Verbose {
		backendStdout := logging.NewLogBackend(os.Stdout, "", 0)
		backendStdoutFormatter = logging.NewBackendFormatter(backendStdout, stdoutLogFormat)
		logging.SetBackend(backendStdoutFormatter)
	}

	if !x.NoLogFiles {
		backendFile := logging.NewLogBackend(w, "", 0)
		backendFileFormatter := logging.NewBackendFormatter(backendFile, fileLogFormat)
		if x.Verbose {
			logging.SetBackend(backendFileFormatter, backendStdoutFormatter)
		} else {
			logging.SetBackend(backendFileFormatter)
		}
		ipfslogging.LdJSONFormatter()
		w2 := &lumberjack.Logger{
			Filename:   path.Join(repoPath, "logs", "ipfs.log"),
			MaxSize:    10, // Megabytes
			MaxBackups: 3,
			MaxAge:     30, // Days
		}
		ipfslogging.Output(w2)()
	}

	var level logging.Level
	switch strings.ToLower(x.LogLevel) {
	case "debug":
		level = logging.DEBUG
	case "info":
		level = logging.INFO
	case "notice":
		level = logging.NOTICE
	case "warning":
		level = logging.WARNING
	case "error":
		level = logging.ERROR
	case "critical":
		level = logging.CRITICAL
	default:
		level = logging.DEBUG
	}
	logging.SetLevel(level, "")

	err = core.CheckAndSetUlimit()
	if err != nil {
		return err
	}

	ct := wi.Bitcoin
	if x.BitcoinCash || strings.Contains(repoPath, "-bitcoincash") {
		ct = wi.BitcoinCash
	} else if x.ZCash != "" || strings.Contains(repoPath, "-zcash") {
		ct = wi.Zcash
	}

	migrations.WalletCoinType = ct
	sqliteDB, err := InitializeRepo(repoPath, x.Password, "", isTestnet, time.Now(), ct)
	if err != nil && err != repo.ErrRepoExists {
		return err
	}

	// Create user-agent file
	userAgentBytes := []byte(core.USERAGENT + x.UserAgent)
	err = ioutil.WriteFile(path.Join(repoPath, "root", "user_agent"), userAgentBytes, os.FileMode(0644))
	if err != nil {
		log.Error("write user_agent:", err)
		return err
	}

	// If the database cannot be decrypted, exit
	if sqliteDB.Config().IsEncrypted() {
		sqliteDB.Close()
		fmt.Print("Database is encrypted, enter your password: ")
		// nolint:unconvert
		bytePassword, _ := terminal.ReadPassword(int(syscall.Stdin))
		fmt.Println("")
		pw := string(bytePassword)
		sqliteDB, err = InitializeRepo(repoPath, pw, "", isTestnet, time.Now(), ct)
		if err != nil && err != repo.ErrRepoExists {
			return err
		}
		if sqliteDB.Config().IsEncrypted() {
			log.Error("Invalid password")
			os.Exit(3)
		}
	}

	// Get creation date. Ignore the error and use a default timestamp.
	creationDate, err := sqliteDB.Config().GetCreationDate()
	if err != nil {
		log.Error("error loading wallet creation date from database - using unix epoch.")
	}

	// Load config
	configFile, err := ioutil.ReadFile(path.Join(repoPath, "config"))
	if err != nil {
		log.Error("read config:", err)
		return err
	}

	apiConfig, err := schema.GetAPIConfig(configFile)
	if err != nil {
		log.Error("scan api config:", err)
		return err
	}
	torConfig, err := schema.GetTorConfig(configFile)
	if err != nil {
		log.Error("scan tor config:", err)
		return err
	}
	dataSharing, err := schema.GetDataSharing(configFile)
	if err != nil {
		log.Error("scan data sharing config:", err)
		return err
	}
	dropboxToken, err := schema.GetDropboxApiToken(configFile)
	if err != nil {
		log.Error("scan dropbox api token:", err)
		return err
	}
	republishInterval, err := schema.GetRepublishInterval(configFile)
	if err != nil {
		log.Error("scan republish interval config:", err)
		return err
	}
	walletsConfig, err := schema.GetWalletsConfig(configFile)
	if err != nil {
		log.Error("scan wallets config:", err)
		return err
	}

	// IPFS node setup
	r, err := fsrepo.Open(repoPath)
	if err != nil {
		log.Error("open repo:", err)
		return err
	}
	cctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := r.Config()
	if err != nil {
		log.Error("get repo config:", err)
		return err
	}

	identityKey, err := sqliteDB.Config().GetIdentityKey()
	if err != nil {
		log.Error("get identity key:", err)
		return err
	}
	identity, err := ipfs.IdentityFromKey(identityKey)
	if err != nil {
		log.Error("get identity from key:", err)
		return err
	}
	cfg.Identity = identity

	// Setup testnet
	if x.Testnet || x.Regtest {
		testnetBootstrapAddrs, err := schema.GetTestnetBootstrapAddrs(configFile)
		if err != nil {
			log.Error(err)
			return err
		}
		cfg.Bootstrap = testnetBootstrapAddrs
		dhtopts.ProtocolDHT = "/openbazaar/kad/testnet/1.0.0"
		bitswap.ProtocolBitswap = "/openbazaar/bitswap/testnet/1.1.0"
		service.ProtocolOpenBazaar = "/openbazaar/app/testnet/1.0.0"

		dataSharing.PushTo = []string{}
	}

	onionAddr, err := obnet.MaybeCreateHiddenServiceKey(repoPath)
	if err != nil {
		log.Error("create onion key:", err)
		return err
	}
	onionAddrString := "/onion/" + onionAddr + ":4003"
	if x.Tor {
		cfg.Addresses.Swarm = []string{}
		cfg.Addresses.Swarm = append(cfg.Addresses.Swarm, onionAddrString)
	} else if x.DualStack {
		cfg.Addresses.Swarm = []string{}
		cfg.Addresses.Swarm = append(cfg.Addresses.Swarm, onionAddrString)
		cfg.Addresses.Swarm = append(cfg.Addresses.Swarm, "/ip4/0.0.0.0/tcp/4001")
		cfg.Addresses.Swarm = append(cfg.Addresses.Swarm, "/ip6/::/tcp/4001")
		cfg.Addresses.Swarm = append(cfg.Addresses.Swarm, "/ip6/::/tcp/9005/ws")
		cfg.Addresses.Swarm = append(cfg.Addresses.Swarm, "/ip4/0.0.0.0/tcp/9005/ws")
	}
	// Iterate over our address and process them as needed
	var onionTransport *oniontp.OnionTransport
	var torDialer proxy.Dialer
	var usingTor, usingClearnet bool
	var controlPort int
	for i, addr := range cfg.Addresses.Swarm {
		m, err := ma.NewMultiaddr(addr)
		if err != nil {
			log.Error("creating swarm multihash:", err)
			return err
		}
		p := m.Protocols()
		// If we are using UTP and the stun option has been select, run stun and replace the port in the address
		if x.STUN && p[0].Name == "ip4" && p[1].Name == "udp" && p[2].Name == "utp" {
			usingClearnet = true
			port, serr := obnet.Stun()
			if serr != nil {
				log.Error("stun setup:", serr)
				return err
			}
			cfg.Addresses.Swarm = append(cfg.Addresses.Swarm[:i], cfg.Addresses.Swarm[i+1:]...)
			cfg.Addresses.Swarm = append(cfg.Addresses.Swarm, "/ip4/0.0.0.0/udp/"+strconv.Itoa(port)+"/utp")
			break
		} else if p[0].Name == "onion" {
			usingTor = true
		} else {
			usingClearnet = true
		}
	}
	// Create Tor transport
	if usingTor {
		torControl := torConfig.TorControl
		if torControl == "" {
			controlPort, err = obnet.GetTorControlPort()
			if err != nil {
				log.Error("get tor control port:", err)
				return err
			}
			torControl = "127.0.0.1:" + strconv.Itoa(controlPort)
		}
		torPw := torConfig.Password
		if x.TorPassword != "" {
			torPw = x.TorPassword
		}
		onionTransport, err = oniontp.NewOnionTransport("tcp4", torControl, torPw, nil, repoPath, (usingTor && usingClearnet))
		if err != nil {
			log.Error("setup tor transport:", err)
			return err
		}
	}
	// If we're only using Tor set the proxy dialer
	if usingTor && !usingClearnet {
		log.Notice("Using Tor exclusively")
		torDialer, err = onionTransport.TorDialer()
		if err != nil {
			log.Error("dailing tor network:", err)
			return err
		}
		cfg.Swarm.DisableNatPortMap = true
	}

	// Custom host option used if Tor is enabled
	defaultHostOption := func(ctx context.Context, id peer.ID, ps pstore.Peerstore, options ...libp2p.Option) (p2phost.Host, error) {
		pkey := ps.PrivKey(id)
		if pkey == nil {
			return nil, fmt.Errorf("missing private key for node ID: %s", id.Pretty())
		}
		options = append([]libp2p.Option{libp2p.Identity(pkey), libp2p.Peerstore(ps)}, options...)
		if usingTor {
			options = append(options, libp2p.Transport(onionTransport.Constructor))
		}
		return libp2p.New(ctx, options...)
	}

	ncfg := &ipfscore.BuildCfg{
		Repo:   r,
		Online: true,
		ExtraOpts: map[string]bool{
			"mplex":  true,
			"ipnsps": true,
		},
		Routing: DHTOption,
	}

	if onionTransport != nil {
		ncfg.Host = defaultHostOption
	}
	nd, err := ipfscore.NewNode(cctx, ncfg)
	if err != nil {
		log.Error("create new ipfs node:", err)
		return err
	}

	ctx := commands.Context{}
	ctx.Online = true
	ctx.ConfigRoot = repoPath
	ctx.LoadConfig = func(_ string) (*config.Config, error) {
		return fsrepo.ConfigAt(repoPath)
	}
	ctx.ConstructNode = func() (*ipfscore.IpfsNode, error) {
		return nd, nil
	}

	log.Info("Peer ID: ", nd.Identity.Pretty())
	printSwarmAddrs(nd)

	// Get current directory root hash
	ipnskey := namesys.IpnsDsKey(nd.Identity)
	ival, hasherr := nd.Repo.Datastore().Get(ipnskey)
	if hasherr != nil {
		log.Error("get ipns key:", hasherr)
	}
	ourIpnsRecord := new(ipnspb.IpnsEntry)
	err = proto.Unmarshal(ival, ourIpnsRecord)
	if err != nil {
		log.Error("unmarshal record value", err)
	}

	// Wallet
	mn, err := sqliteDB.Config().GetMnemonic()
	if err != nil {
		log.Error("get config mnemonic:", err)
		return err
	}
	var params chaincfg.Params
	if x.Testnet {
		params = chaincfg.TestNet3Params
	} else if x.Regtest {
		params = chaincfg.RegressionNetParams
	} else {
		params = chaincfg.MainNetParams
	}

	// Multiwallet setup
	var walletLogWriter io.Writer
	if x.NoLogFiles {
		walletLogWriter = &DummyWriter{}
	} else {
		walletLogWriter = &lumberjack.Logger{
			Filename:   path.Join(repoPath, "logs", "wallet.log"),
			MaxSize:    10, // Megabytes
			MaxBackups: 3,
			MaxAge:     30, // Days
		}
	}
	walletLogFile := logging.NewLogBackend(walletLogWriter, "", 0)
	walletFileFormatter := logging.NewBackendFormatter(walletLogFile, fileLogFormat)
	walletLogger := logging.MultiLogger(walletFileFormatter)
	multiwalletConfig := &wallet.WalletConfig{
		ConfigFile:           walletsConfig,
		DB:                   sqliteDB.DB(),
		Params:               &params,
		RepoPath:             repoPath,
		Logger:               walletLogger,
		Proxy:                torDialer,
		WalletCreationDate:   creationDate,
		Mnemonic:             mn,
		DisableExchangeRates: x.DisableExchangeRates,
	}
	mw, err := wallet.NewMultiWallet(multiwalletConfig)
	if err != nil {
		return err
	}
	resyncManager := resync.NewResyncManager(sqliteDB.Sales(), mw)

	// Master key setup
	seed := bip39.NewSeed(mn, "")
	mPrivKey, err := hdkeychain.NewMaster(seed, &params)
	if err != nil {
		log.Error(err)
		return err
	}

	// Push nodes
	var pushNodes []peer.ID
	for _, pnd := range dataSharing.PushTo {
		p, err := peer.IDB58Decode(pnd)
		if err != nil {
			log.Error("Invalid peerID in DataSharing config")
			return err
		}
		pushNodes = append(pushNodes, p)
	}

	// Authenticated gateway
	gatewayMaddr, err := ma.NewMultiaddr(cfg.Addresses.Gateway[0])
	if err != nil {
		log.Error(err)
		return err
	}
	addr, err := gatewayMaddr.ValueForProtocol(ma.P_IP4)
	if err != nil {
		log.Error(err)
		return err
	}
	// Override config file preference if this is Mainnet, open internet and API enabled
	if addr != "127.0.0.1" && params.Name == chaincfg.MainNetParams.Name && apiConfig.Enabled {
		apiConfig.Authenticated = true
	}
	apiConfig.AllowedIPs = append(apiConfig.AllowedIPs, x.AllowIP...)

	// Create authentication cookie
	var authCookie http.Cookie
	authCookie.Name = "OpenBazaar_Auth_Cookie"

	if x.AuthCookie != "" {
		authCookie.Value = x.AuthCookie
		apiConfig.Authenticated = true
	} else {
		cookiePrefix := authCookie.Name + "="
		cookiePath := path.Join(repoPath, ".cookie")
		cookie, err := ioutil.ReadFile(cookiePath)
		if err != nil {
			authBytes := make([]byte, 32)
			_, err = rand.Read(authBytes)
			if err != nil {
				log.Error(err)
				return err
			}
			authCookie.Value = base58.Encode(authBytes)
			f, err := os.Create(cookiePath)
			if err != nil {
				log.Error(err)
				return err
			}
			cookie := cookiePrefix + authCookie.Value
			_, werr := f.Write([]byte(cookie))
			if werr != nil {
				log.Error(werr)
				return werr
			}
			f.Close()
		} else {
			if string(cookie)[:len(cookiePrefix)] != cookiePrefix {
				return errors.New("Invalid authentication cookie. Delete it to generate a new one")
			}
			split := strings.SplitAfter(string(cookie), cookiePrefix)
			authCookie.Value = split[1]
		}
	}

	// Set up the ban manager
	settings, err := sqliteDB.Settings().Get()
	if err != nil && err != db.SettingsNotSetError {
		log.Error(err)
		return err
	}
	var blockedNodes []peer.ID
	if settings.BlockedNodes != nil {
		for _, pid := range *settings.BlockedNodes {
			id, err := peer.IDB58Decode(pid)
			if err != nil {
				continue
			}
			blockedNodes = append(blockedNodes, id)
		}
	}
	bm := obnet.NewBanManager(blockedNodes)

	if x.Testnet {
		setTestmodeRecordAgingIntervals()
	}

	// Build pubsub
	publisher := ipfs.NewPubsubPublisher(context.Background(), nd.PeerHost, nd.Routing, nd.Repo.Datastore(), nd.PubSub)
	subscriber := ipfs.NewPubsubSubscriber(context.Background(), nd.PeerHost, nd.Routing, nd.Repo.Datastore(), nd.PubSub)
	ps := ipfs.Pubsub{Publisher: publisher, Subscriber: subscriber}

	// OpenBazaar node setup
	core.Node = &core.OpenBazaarNode{
		AcceptStoreRequests:           dataSharing.AcceptStoreRequests,
		BanManager:                    bm,
		Datastore:                     sqliteDB,
		IPNSBackupAPI:                 "", // TODO [cp]: need a migration to set this field in another location.
		IpfsNode:                      nd,
		MasterPrivateKey:              mPrivKey,
		Multiwallet:                   mw,
		OfflineMessageFailoverTimeout: 30 * time.Second,
		Pubsub:               ps,
		PushNodes:            pushNodes,
		RegressionTestEnable: x.Regtest,
		RepoPath:             repoPath,
		RootHash:             string(ourIpnsRecord.Value),
		TestnetEnable:        x.Testnet,
		TorDialer:            torDialer,
		UserAgent:            core.USERAGENT,
	}
	core.PublishLock.Lock()

	// Offline messaging storage
	var storage sto.OfflineMessagingStorage
	if x.Storage == "self-hosted" || x.Storage == "" {
		storage = selfhosted.NewSelfHostedStorage(repoPath, core.Node.IpfsNode, pushNodes, core.Node.SendStore)
	} else if x.Storage == "dropbox" {
		if usingTor && !usingClearnet {
			log.Error("Dropbox can not be used with Tor")
			return errors.New("Dropbox can not be used with Tor")
		}

		if dropboxToken == "" {
			err = errors.New("Dropbox token not set in config file")
			log.Error(err)
			return err
		}
		storage, err = dropbox.NewDropBoxStorage(dropboxToken)
		if err != nil {
			log.Error(err)
			return err
		}
	} else {
		err = errors.New("Invalid storage option")
		log.Error(err)
		return err
	}
	core.Node.MessageStorage = storage

	if len(cfg.Addresses.Gateway) <= 0 {
		return ErrNoGateways
	}
	if (apiConfig.SSL && apiConfig.SSLCert == "") || (apiConfig.SSL && apiConfig.SSLKey == "") {
		return errors.New("SSL cert and key files must be set when SSL is enabled")
	}

	gateway, err := newHTTPGateway(core.Node, ctx, authCookie, *apiConfig, x.NoLogFiles)
	if err != nil {
		log.Error(err)
		return err
	}

	if len(cfg.Addresses.API) > 0 && cfg.Addresses.API[0] != "" {
		if _, err := serveHTTPApi(&ctx); err != nil {
			log.Error(err)
			return err
		}
	}

	go func() {
		if !x.DisableWallet {
			// If the wallet doesn't allow resyncing from a specific height to scan for unpaid orders, wait for all messages to process before continuing.
			if resyncManager == nil {
				core.Node.WaitForMessageRetrieverCompletion()
			}
			TL := lis.NewTransactionListener(core.Node.Multiwallet, core.Node.Datastore, core.Node.Broadcast)
			for ct, wal := range mw {
				WL := lis.NewWalletListener(core.Node.Datastore, core.Node.Broadcast, ct)
				wal.AddTransactionListener(WL.OnTransactionReceived)
				wal.AddTransactionListener(TL.OnTransactionReceived)
			}
			log.Info("Starting multiwallet...")
			su := wallet.NewStatusUpdater(mw, core.Node.Broadcast, nd.Context())
			go su.Start()
			go mw.Start()
			if resyncManager != nil {
				go resyncManager.Start()
				go func() {
					core.Node.WaitForMessageRetrieverCompletion()
					resyncManager.CheckUnfunded()
				}()
			}
		}
		core.Node.Service = service.New(core.Node, sqliteDB)

		core.Node.StartMessageRetriever()
		core.Node.StartPointerRepublisher()
		core.Node.StartRecordAgingNotifier()

		core.PublishLock.Unlock()
		err = core.Node.UpdateFollow()
		if err != nil {
			log.Error(err)
		}
		if !core.InitalPublishComplete {
			err = core.Node.SeedNode()
			if err != nil {
				log.Error(err)
			}
		}
		core.Node.SetUpRepublisher(republishInterval)
	}()

	// Start gateway
	err = gateway.Serve()
	if err != nil {
		log.Error(err)
	}

	return nil
}

func setTestmodeRecordAgingIntervals() {
	repo.VendorDisputeTimeout_lastInterval = time.Duration(60) * time.Minute

	repo.ModeratorDisputeExpiry_firstInterval = time.Duration(20) * time.Minute
	repo.ModeratorDisputeExpiry_secondInterval = time.Duration(40) * time.Minute
	repo.ModeratorDisputeExpiry_thirdInterval = time.Duration(59) * time.Minute
	repo.ModeratorDisputeExpiry_lastInterval = time.Duration(60) * time.Minute

	repo.BuyerDisputeTimeout_firstInterval = time.Duration(20) * time.Minute
	repo.BuyerDisputeTimeout_secondInterval = time.Duration(40) * time.Minute
	repo.BuyerDisputeTimeout_thirdInterval = time.Duration(59) * time.Minute
	repo.BuyerDisputeTimeout_lastInterval = time.Duration(60) * time.Minute
	repo.BuyerDisputeTimeout_totalDuration = time.Duration(60) * time.Minute

	repo.BuyerDisputeExpiry_firstInterval = time.Duration(20) * time.Minute
	repo.BuyerDisputeExpiry_secondInterval = time.Duration(40) * time.Minute
	repo.BuyerDisputeExpiry_lastInterval = time.Duration(59) * time.Minute
	repo.BuyerDisputeExpiry_totalDuration = time.Duration(60) * time.Minute
}

// Prints the addresses of the host
func printSwarmAddrs(node *ipfscore.IpfsNode) {
	var addrs []string
	for _, addr := range node.PeerHost.Addrs() {
		addrs = append(addrs, addr.String())
	}
	sort.Strings(addrs)

	for _, addr := range addrs {
		log.Infof("Swarm listening on %s\n", addr)
	}
}

type DummyWriter struct{}

func (d *DummyWriter) Write(p []byte) (n int, err error) {
	return 0, nil
}

type DummyListener struct {
	addr net.Addr
}

func (d *DummyListener) Addr() net.Addr {
	return d.addr
}

func (d *DummyListener) Accept() (net.Conn, error) {
	conn, _ := net.FileConn(nil)
	return conn, nil
}

func (d *DummyListener) Close() error {
	return nil
}

// Collects options, creates listener, prints status message and starts serving requests
func newHTTPGateway(node *core.OpenBazaarNode, ctx commands.Context, authCookie http.Cookie, config schema.APIConfig, noLogFiles bool) (*api.Gateway, error) {
	// Get API configuration
	cfg, err := ctx.GetConfig()
	if err != nil {
		return nil, err
	}

	// Create a network listener
	gatewayMaddr, err := ma.NewMultiaddr(cfg.Addresses.Gateway[0])
	if err != nil {
		return nil, fmt.Errorf("newHTTPGateway: invalid gateway address: %q (err: %s)", cfg.Addresses.Gateway, err)
	}
	var gwLis manet.Listener
	if config.SSL {
		netAddr, err := manet.ToNetAddr(gatewayMaddr)
		if err != nil {
			return nil, err
		}
		gwLis, err = manet.WrapNetListener(&DummyListener{netAddr})
		if err != nil {
			return nil, err
		}
	} else {
		gwLis, err = manet.Listen(gatewayMaddr)
		if err != nil {
			return nil, fmt.Errorf("newHTTPGateway: manet.Listen(%s) failed: %s", gatewayMaddr, err)
		}
	}

	// We might have listened to /tcp/0 - let's see what we are listing on
	gatewayMaddr = gwLis.Multiaddr()
	log.Infof("Gateway/API server listening on %s\n", gatewayMaddr)

	// Setup an options slice
	var opts = []corehttp.ServeOption{
		corehttp.MetricsCollectionOption("gateway"),
		corehttp.CommandsROOption(ctx),
		corehttp.VersionOption(),
		corehttp.IPNSHostnameOption(),
		corehttp.GatewayOption(cfg.Gateway.Writable, "/ipfs", "/ipns"),
	}

	if len(cfg.Gateway.RootRedirect) > 0 {
		opts = append(opts, corehttp.RedirectOption("", cfg.Gateway.RootRedirect))
	}

	if err != nil {
		return nil, fmt.Errorf("newHTTPGateway: ConstructNode() failed: %s", err)
	}

	// Create and return an API gateway
	var w4 io.Writer
	if noLogFiles {
		w4 = &DummyWriter{}
	} else {
		w4 = &lumberjack.Logger{
			Filename:   path.Join(node.RepoPath, "logs", "api.log"),
			MaxSize:    10, // Megabytes
			MaxBackups: 3,
			MaxAge:     30, // Days
		}
	}
	apiFile := logging.NewLogBackend(w4, "", 0)
	apiFileFormatter := logging.NewBackendFormatter(apiFile, fileLogFormat)
	ml := logging.MultiLogger(apiFileFormatter)

	return api.NewGateway(node, authCookie, manet.NetListener(gwLis), config, ml, opts...)
}

var DHTOption ipfscore.RoutingOption = constructDHTRouting

const IpnsValidatorTag = "ipns"

func constructDHTRouting(ctx context.Context, host p2phost.Host, dstore ds.Batching, validator record.Validator) (routing.IpfsRouting, error) {
	return dht.New(
		ctx, host,
		dhtopts.Datastore(dstore),
		dhtopts.Validator(validator),
	)
}

// serveHTTPApi collects options, creates listener, prints status message and starts serving requests
func serveHTTPApi(cctx *commands.Context) (<-chan error, error) {
	cfg, err := cctx.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("serveHTTPApi: GetConfig() failed: %s", err)
	}

	apiAddr := cfg.Addresses.API[0]
	apiMaddr, err := ma.NewMultiaddr(apiAddr)
	if err != nil {
		return nil, fmt.Errorf("serveHTTPApi: invalid API address: %q (err: %s)", apiAddr, err)
	}

	apiLis, err := manet.Listen(apiMaddr)
	if err != nil {
		return nil, fmt.Errorf("serveHTTPApi: manet.Listen(%s) failed: %s", apiMaddr, err)
	}
	// we might have listened to /tcp/0 - lets see what we are listing on
	apiMaddr = apiLis.Multiaddr()
	fmt.Printf("API server listening on %s\n", apiMaddr)

	// by default, we don't let you load arbitrary ipfs objects through the api,
	// because this would open up the api to scripting vulnerabilities.
	// only the webui objects are allowed.
	// if you know what you're doing, go ahead and pass --unrestricted-api.
	unrestricted := false
	gatewayOpt := corehttp.GatewayOption(false, corehttp.WebUIPaths...)
	if unrestricted {
		gatewayOpt = corehttp.GatewayOption(true, "/ipfs", "/ipns")
	}

	var opts = []corehttp.ServeOption{
		corehttp.MetricsCollectionOption("api"),
		corehttp.CommandsOption(*cctx),
		corehttp.WebUIOption,
		gatewayOpt,
		corehttp.VersionOption(),
		corehttp.MetricsScrapingOption("/debug/metrics/prometheus"),
		corehttp.LogOption(),
	}

	if len(cfg.Gateway.RootRedirect) > 0 {
		opts = append(opts, corehttp.RedirectOption("", cfg.Gateway.RootRedirect))
	}

	node, err := cctx.ConstructNode()
	if err != nil {
		return nil, fmt.Errorf("serveHTTPApi: ConstructNode() failed: %s", err)
	}

	if err := node.Repo.SetAPIAddr(apiMaddr); err != nil {
		return nil, fmt.Errorf("serveHTTPApi: SetAPIAddr() failed: %s", err)
	}

	errc := make(chan error)
	go func() {
		errc <- corehttp.Serve(node, manet.NetListener(apiLis), opts...)
		close(errc)
	}()
	return errc, nil
}

func InitializeRepo(dataDir, password, mnemonic string, testnet bool, creationDate time.Time, coinType wi.CoinType) (*db.SQLiteDatastore, error) {
	// Database
	sqliteDB, err := db.Create(dataDir, password, testnet, coinType)
	if err != nil {
		return sqliteDB, err
	}

	// Initialize the IPFS repo if it does not already exist
	err = repo.DoInit(dataDir, 4096, testnet, password, mnemonic, creationDate, sqliteDB.Config().Init)
	if err != nil {
		return sqliteDB, err
	}
	return sqliteDB, nil
}

func printSplashScreen(verbose bool) {
	blue := color.New(color.FgBlue)
	white := color.New(color.FgWhite)

	for i, l := range []string{
		"________             ",
		"         __________",
		`\_____  \ ______   ____   ____`,
		`\______   \_____  _____________  _____ _______`,
		` /   |   \\____ \_/ __ \ /    \`,
		`|    |  _/\__  \ \___   /\__  \ \__  \\_  __ \ `,
		`/    |    \  |_> >  ___/|   |  \    `,
		`|   \ / __ \_/    /  / __ \_/ __ \|  | \/`,
		`\_______  /   __/ \___  >___|  /`,
		`______  /(____  /_____ \(____  (____  /__|`,
		`        \/|__|        \/     \/  `,
		`     \/      \/      \/     \/     \/`,
	} {
		if i%2 == 0 {
			if _, err := white.Printf(l); err != nil {
				log.Debug(err)
				return
			}
			continue
		}
		if _, err := blue.Println(l); err != nil {
			log.Debug(err)
			return
		}
	}

	blue.DisableColor()
	white.DisableColor()
	fmt.Println("")
	fmt.Println("OpenBazaar Server v" + core.VERSION)
	if !verbose {
		fmt.Println("[Press Ctrl+C to exit]")
	}
}
