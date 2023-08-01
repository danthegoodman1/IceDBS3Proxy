import { ListObjectsV2Command, S3Client } from "@aws-sdk/client-s3";

const clientVhost = new S3Client({
  endpoint: "http://localhost:8080",
  region: "us-east-1",
  credentials: {
    accessKeyId: "blah",
    secretAccessKey: "blah"
  }
})

const clientPath = new S3Client({
  endpoint: "http://localhost:8080",
  region: "us-east-1",
  forcePathStyle: true,
  credentials: {
    accessKeyId: "blah",
    secretAccessKey: "blah"
  }
})

const command = new ListObjectsV2Command({
  Bucket: "testbucket",
  Prefix: "hey/",
})

const res = await clientPath.send(command)
console.log(res)
console.log(`got ${res.Contents?.length} items`)
