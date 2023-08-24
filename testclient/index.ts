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
  MaxKeys: 123,
}))
console.log(`got ${res.Contents?.length} items`)
const target = res.Contents![0].Key!
console.log('getting file:', target )

console.log("checking path style routing get object")
const getRes = await clientPath.send(new GetObjectCommand({
  Bucket: "testbucket",
  Key: target,
}))
const content = await getRes.Body?.transformToString()
console.log(content, content?.length, "bytes")
