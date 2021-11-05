#!/bin/bash

if [ -z "$RESOURCE_GROUP" ]; then
    echo "must provide a RESOURCE_GROUP env var"
    exit 1;
fi

if [ -z "$REGION" ]; then
    echo "must provide a REGION env var"
    exit 1;
fi

if [ -z "$NAME" ]; then
    echo "must provide a NAME env var"
    exit 1;
fi

export AKS_IAAS_RESOURCE_GROUP="MC_${RESOURCE_GROUP}_${NAME}_${REGION}"

# Continually look for non-Succeeded VMSS instances
vmssHealthCheck() {
  while true; do
    NUM_VMSS=0
    NUM_TERMINAL_VMSS=0
    echo "$(date)    Starting VMSS Health Remediation loop"
    for VMSS in $(az vmss list -g $AKS_IAAS_RESOURCE_GROUP | jq -r '.[] | .name'); do
      ((NUM_VMSS++))
      NUM_DELETED_INSTANCES=0
      VMSS_PROVISIONING_STATE=$(az vmss show -g $AKS_IAAS_RESOURCE_GROUP -n $VMSS | jq -r '.provisioningState')
      echo $(date)    VMSS $VMSS has a ProvisioningState of $VMSS_PROVISIONING_STATE
      VMSS_CAPACITY=$(az vmss list -g $AKS_IAAS_RESOURCE_GROUP | jq -r --arg VMSS "$VMSS" '.[] | select(.name == $VMSS) | .sku.capacity')
      echo $(date)    VMSS $VMSS has a current capacity of $VMSS_CAPACITY
      if [ "$VMSS_PROVISIONING_STATE" == "Succeeded" ] || [ "$VMSS_PROVISIONING_STATE" == "Failed" ]; then
        ((NUM_TERMINAL_VMSS++))
        HAS_FAILED_STATE_INSTANCE="false"
        for TARGET_VMSS_INSTANCE in $(az vmss list-instances -g $AKS_IAAS_RESOURCE_GROUP -n $VMSS | jq -r '.[] | select(.provisioningState == "Failed") | .name'); do
          HAS_FAILED_STATE_INSTANCE="true"
          echo $(date)    Deleting VMSS $VMSS instance $TARGET_VMSS_INSTANCE
          if ! az vmss delete-instances -n $VMSS -g $AKS_IAAS_RESOURCE_GROUP --instance-id ${TARGET_VMSS_INSTANCE##*_} --no-wait; then
            sleep 30
          else
            sleep 1
            ((NUM_DELETED_INSTANCES++))
          fi
        done
        for TARGET_VMSS_INSTANCE in $(az vmss list-instances -g $AKS_IAAS_RESOURCE_GROUP -n $VMSS | jq -r '.[].resources[] | select(.name == "vmssCSE" and .provisioningState == "Failed") | .id' | awk -F'/' '{print $9}'); do
          HAS_FAILED_STATE_INSTANCE="true"
          echo $(date)    Deleting VMSS $VMSS instance $TARGET_VMSS_INSTANCE
          if ! az vmss delete-instances -n $VMSS -g $AKS_IAAS_RESOURCE_GROUP --instance-id ${TARGET_VMSS_INSTANCE##*_}; then
            sleep 30
          else
            sleep 1
            ((NUM_DELETED_INSTANCES++))
          fi
        done
        for TARGET_VMSS_INSTANCE in $(az vmss list-instances -g $AKS_IAAS_RESOURCE_GROUP -n $VMSS | jq -r '.[].resources[] | select(.publisher == "Microsoft.AKS" and .provisioningState != "Succeeded" and .provisioningState != "Creating" and .provisioningState != "Deleting") | .id' | awk -F'/' '{print $9}'); do
          HAS_FAILED_STATE_INSTANCE="true"
          echo $(date)    Deleting VMSS $VMSS instance $TARGET_VMSS_INSTANCE
          if ! az vmss delete-instances -n $VMSS -g $AKS_IAAS_RESOURCE_GROUP --instance-id ${TARGET_VMSS_INSTANCE##*_}; then
            sleep 30
          else
            sleep 1
            ((NUM_DELETED_INSTANCES++))
          fi
        done
        if [ "$HAS_FAILED_STATE_INSTANCE" == "true" ]; then
          echo $(date)    Waiting for $VMSS to reach a terminal ProvisioningState after failed instances were deleted...
          sleep 30
          until [[ $(az vmss show -g $AKS_IAAS_RESOURCE_GROUP -n $VMSS | jq -r '. | select(.provisioningState == "Succeeded" or .provisioningState == "Failed") | .name') ]]; do
            echo $(date)    Waiting for $VMSS to reach a terminal ProvisioningState after failed instances were deleted...
            sleep 30
          done
          echo $(date)    VMSS $VMSS is in a terminal state after failed instances were deleted!
        fi
      fi
      if [ "$NUM_DELETED_INSTANCES" -gt "0" ]; then
        echo $(date)    Instances were deleted from VMSS $VMSS, ensuring that capacity is set to $VMSS_CAPACITY
        az vmss scale --new-capacity $VMSS_CAPACITY -n $VMSS -g $AKS_IAAS_RESOURCE_GROUP --no-wait;
      fi
    done
    sleep 150
  done
}

vmssHealthCheck &
VMSS_HEALTH_CHECK_PID=$!

cleanup() {
    kill $VMSS_HEALTH_CHECK_PID >/dev/null 2>&1
    exit 0
}

trap cleanup SIGINT

while true; do
  sleep 30
done
