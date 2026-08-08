Status: DONE
Commit: c2fc9306ac397c4cfc38bdeaf8e667b91f1b2b44
RED: UploadImage undefined on Mock/HTTP/Client; unknown Uploader on CaseSnapshot
GREEN: go test comfyui + actuator ok; NewClient mock/HTTP still pass
Files: comfyui client/http/upload_test; actuator snapshot + tests
Risk: cross-module, diff>200; mime not set on multipart Content-Type (noted)
Notes: ImageUploader on CaseSnapshot; image path reads .blob.json then Upload then write filename
