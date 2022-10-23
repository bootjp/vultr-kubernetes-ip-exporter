package main

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/vultr/govultr/v2"

	"golang.org/x/oauth2"
	"log"
	"net"
	"os"
)

func init() {
	var ok bool
	apiKey, ok = os.LookupEnv("VULTR_TOKEN")
	if !ok {
		log.Fatal("Could not find VULTR_TOKEN, please assert it is set.")
	}
	kid, ok = os.LookupEnv("KUBERNETES_CLUSTER_ID")
	if !ok {
		log.Fatal("Could not find KUBERNETES_CLUSTER_ID, please assert it is set.")
	}

	nid, ok = os.LookupEnv("NODE_POOL_ID")
	if !ok {
		log.Fatal("Could not find NODE_POOL_ID, please assert it is set.")
	}

	redisHost, ok = os.LookupEnv("REDIS_HOST")
	if !ok {
		log.Fatal("Could not find REDIS_HOST, please assert it is set.")
	}

	redisPort, ok = os.LookupEnv("REDIS_PORT")
	if !ok {
		log.Fatal("Could not find REDIS_PORT, please assert it is set.")
	}

	redisPassword, ok = os.LookupEnv("REDIS_PASSWORD")
	if !ok {
		log.Println("Redis Password is not set, assuming no password")
	}

	redisKey, ok = os.LookupEnv("REDIS_KEY")
	if !ok {
		log.Fatal("Could not find REDIS_KEY, please assert it is set.")
	}

}

var kid string
var nid string
var apiKey string
var redisPort string
var redisHost string
var redisPassword string
var redisKey string

func main() {
	ctx := context.Background()
	ips, err := fetchIPs(ctx)
	if err != nil {
		log.Fatal(err)
	}
	cidr, err := convertIP2Cidr(ips)
	if err != nil {
		log.Fatal(err)
	}

	for _, ipNet := range cidr {
		fmt.Println(ipNet.String())
	}

	saveRedis(ctx, cidr)

}

func saveRedis(ctx context.Context, cidr []net.IPNet) error {
	rdb := redis.NewClient(&redis.Options{
		Addr:     net.JoinHostPort(redisHost, redisPort),
		Password: redisPassword,
		DB:       0,
	})

	pipe := rdb.TxPipeline()

	pipe.Del(ctx, redisKey)
	for _, ipNet := range cidr {
		_, err := pipe.SAdd(ctx, redisKey, ipNet.String()).Result()
		if err != nil {
			return err
		}
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		return err
	}
	return nil

}

func fetchIPs(ctx context.Context) ([]net.IP, error) {
	config := &oauth2.Config{}
	ts := config.TokenSource(ctx, &oauth2.Token{AccessToken: apiKey})
	vultrClient := govultr.NewClient(oauth2.NewClient(ctx, ts))
	nodes, err := vultrClient.Kubernetes.GetNodePool(ctx, kid, nid)
	if err != nil {
		return nil, err
	}

	var ips []net.IP

	for _, node := range nodes.Nodes {
		r, err := vultrClient.Instance.Get(ctx, node.ID)
		if err != nil {
			return nil, err
		}
		ips = append(ips, net.ParseIP(r.MainIP))
	}

	return ips, nil
}

func convertIP2Cidr(ips []net.IP) ([]net.IPNet, error) {
	var ipns []net.IPNet

	for _, ip := range ips {
		_, ipn, err := net.ParseCIDR(ip.String() + "/32")
		if err != nil {
			return nil, err
		}
		ipns = append(ipns, *ipn)
	}

	return ipns, nil
}
