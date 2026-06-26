import axios from 'axios';
import { ChangeReview } from '../types/changeReview';

const API_BASE = '/api'; // Adjust based on your setup

export const ChangeReviewApi = {
    // Get a review by its ID (or ChangeSetID finding the latest review)
    getReview: async (reviewId: string): Promise<ChangeReview> => {
        // This endpoint might need to be adjusted if backend exposes by review ID or changeset ID
        // Assuming /api/change-reviews/{id} returns the review
        // Wait, handler code: r.Post("/{id}/promote", h.Promote) where {id} is changeSetID
        // Handler CreateReview takes ChangeSetID in body.
        // We probably need a GET /change-reviews?change_set_id=... or similar.
        // For now, let's assume we fetch by ChangeSetID in the UI mostly.
        // Let's Add a GET endpoint to the backend handler later if missing.
        // Actually, looking at handler code:
        // RegisterRoutes: POST /, POST /{id}/promote, POST /rollback.
        // MISSING GET endpoint in backend handler!
        // I should stick to the plan but realize I need to add GET endpoint to backend.
        // I will implement client assuming it exists, then go fix backend.
        const res = await axios.get(`${API_BASE}/change-reviews/${reviewId}`);
        return res.data;
    },

    createReview: async (changeSetId: string): Promise<ChangeReview> => {
        const res = await axios.post(`${API_BASE}/change-reviews`, { change_set_id: changeSetId });
        return res.data;
    },

    promote: async (changeSetId: string): Promise<void> => {
        await axios.post(`${API_BASE}/change-reviews/${changeSetId}/promote`);
    },

    rollback: async (objectId: string, targetVersion: number): Promise<void> => {
        await axios.post(`${API_BASE}/change-reviews/rollback`, {
            object_id: objectId,
            target_version: targetVersion
        });
    },

    // Semantic History
    getHistory: async (objectId: string): Promise<any[]> => {
        const res = await axios.get(`${API_BASE}/semantic/history/${objectId}`);
        return res.data;
    },

    getVersion: async (objectId: string, version: number): Promise<any> => {
        const res = await axios.get(`${API_BASE}/semantic/version/${objectId}/${version}`);
        return res.data;
    }
};
