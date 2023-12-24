import express, {Request} from 'express'

const app = express()
app.use(express.json())

interface VirtualBucket {
    Prefix: string
    TimeMS: number
}

app.post('/resolve_virtual_bucket', async (req: Request<{}, VirtualBucket, {
    VirtualBucket: string
    KeyID: string
}>, res) => {
    console.log('resolving virtual bucket', req.body.VirtualBucket, req.body.KeyID)
    res.json({
        Prefix: 'example',
        TimeMS: new Date().getTime()
    } as VirtualBucket)
})

app.listen('8888', () => {
    console.log('listening on port 8888')
})