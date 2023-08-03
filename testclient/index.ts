import { GetObjectCommand, ListObjectsV2Command, S3Client } from "@aws-sdk/client-s3";

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

console.log("checking path style routing list")
let res = await clientPath.send(new ListObjectsV2Command({
  Bucket: "testbucket",
  Prefix: "hey/",
  MaxKeys: 123
}))
console.log(`got ${res.Contents?.length} items`)

console.log("checking path style routing get object")
const getRes = await clientPath.send(new GetObjectCommand({
  Bucket: "testbucket",
  Key: "twitch_extensions.csv"
}))
console.log((await getRes.Body?.transformToString())?.length, "bytes")
