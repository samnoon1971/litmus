package self_deployer

import (
	"encoding/json"
	"log"
	"strings"

	"github.com/litmuschaos/litmus/litmus-portal/graphql-server/utils"

	"github.com/litmuschaos/litmus/litmus-portal/graphql-server/pkg/cluster"
	clusterHandler "github.com/litmuschaos/litmus/litmus-portal/graphql-server/pkg/cluster/handler"

	"github.com/litmuschaos/litmus/litmus-portal/graphql-server/pkg/k8s"

	"github.com/litmuschaos/litmus/litmus-portal/graphql-server/graph/model"
)

// StartDeployer registers a new internal self-cluster and starts the deployer
func StartDeployer(projectID string) {
	var (
		isAllManifestInstall  = true
		deployerNamespace     = utils.Config.AgentNamespace
		agentScope            = utils.Config.AgentScope
		skipSSL               = utils.Config.SkipSslVerify
		selfAgentNodeSelector = utils.Config.SelfAgentNodeSelector
		selfAgentTolerations  = utils.Config.SelfAgentTolerations

		failedManifest string
	)

	tolerations := []*model.Toleration{}
	nodeSelector := &selfAgentNodeSelector

	if selfAgentNodeSelector == "" {
		nodeSelector = nil
	}

	if selfAgentTolerations != "" {
		err := json.Unmarshal([]byte(selfAgentTolerations), &tolerations)
		if err != nil {
			log.Print("SELF CLUSTER REG FAILED[TOLERATION-PARSING] : ", err)
			// if toleration parsing fails skip actual manifest apply
			return
		}
	} else {
		tolerations = nil
	}

	clusterInput := model.RegisterClusterRequest{
		ProjectID:      projectID,
		ClusterName:    "Self-Agent",
		ClusterType:    "internal",
		PlatformName:   "others",
		AgentScope:     agentScope,
		AgentNamespace: &deployerNamespace,
		NodeSelector:   nodeSelector,
		Tolerations:    tolerations,
	}

	if strings.ToLower(skipSSL) == "true" {
		skip := true
		clusterInput.SkipSsl = &skip
	}

	resp, err := clusterHandler.RegisterCluster(clusterInput)
	if err != nil {
		log.Print("SELF CLUSTER REG FAILED[DB-REG] : ", err)
		// if cluster registration fails skip actual manifest apply
		return
	}

	response, statusCode, err := cluster.GetManifest(resp.Token)
	if err != nil {
		log.Print("ERROR", err)
	}

	if statusCode == 200 {
		manifests := strings.Split(string(response), "---")
		for _, manifest := range manifests {
			if len(strings.TrimSpace(manifest)) > 0 {
				_, err = k8s.ClusterResource(manifest, deployerNamespace)
				if err != nil {
					log.Print(err)
					failedManifest = failedManifest + manifest
					isAllManifestInstall = false
				}
			}

		}
	}

	if isAllManifestInstall == true {
		log.Print("ALL MANIFESTS HAS BEEN INSTALLED:")
	} else {
		log.Print("SOME MANIFESTS HAS NOT BEEN INSTALLED:", failedManifest)
	}
}
