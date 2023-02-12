CONNECTION_STRING="host=localhost port=5432"
gcloud run deploy --source=. \
--service-account=$SA gorm-test \
--set-env-vars=PROJECT_ID=$GOOGLE_CLOUD_PROJECT,INSTANCE_NAME=test-instance,DATABASE_NAME=music,CONNECTION_STRING="$CONNECTION_STRING" \
--region=us-east5 \
--cpu=1 --memory=2Gi
