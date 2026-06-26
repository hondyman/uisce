import express from 'express';
import { resolveHoldingMarketValue } from './holdingResolver';

const app = express();
app.use(express.json());

app.post('/resolve/holding', async (req, res) => {
    try {
        const result = await resolveHoldingMarketValue(req.body);
        res.json(result);
    } catch (err: any) {
        console.error(err);
        res.status(500).json({ error: err.message });
    }
});

const port = process.env.PORT || 9003;
app.listen(port, () => console.log(`Resolver listening on ${port}`));
