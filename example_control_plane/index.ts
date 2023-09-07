import express, {Request} from 'express'

const app = express()
app.use(express.json())

interface VirtualBucket {
    Prefix: string
    TimeMS: number
}

app.post('/resolve_virtual_bucket', async (req: Request<{}, VirtualBucket, {
    VirtualBucket: string
}>, res) => {
    console.log('resolving virtual bucket', req.body.VirtualBucket)
    res.json({
        Prefix: 'namespaces/user_abc',
        TimeMS: new Date().getTime()
    } as VirtualBucket)
})

app.listen('8888', () => {
    console.log('listening on port 8888')
})