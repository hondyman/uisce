import { useState, useEffect } from 'react';
import { listComments, addComment } from './api';
import type { Comment } from './types';

interface CommentsPanelProps {
  assetId: string;
  assetType: 'query' | 'workbook';
}

function CommentThread({ comment, replies }: { comment: Comment, replies: Comment[] }) {
  return (
    <div className="comment-thread">
      <div className="comment">
        <strong>{comment.author_user_id}</strong>
        <p>{comment.body}</p>
        <small>{new Date(comment.created_at).toLocaleString()}</small>
      </div>
      {replies.map(reply => (
        <div key={reply.id} className="comment reply">
          <strong>{reply.author_user_id}</strong>
          <p>{reply.body}</p>
          <small>{new Date(reply.created_at).toLocaleString()}</small>
        </div>
      ))}
    </div>
  );
}

export default function CommentsPanel({ assetId, assetType }: CommentsPanelProps) {
  const [comments, setComments] = useState<Comment[]>([]);
  const [newComment, setNewComment] = useState('');
  const [loading, setLoading] = useState(true);

  const fetchComments = () => {
    setLoading(true);
    listComments(assetId).then(setComments).finally(() => setLoading(false));
  };

  useEffect(fetchComments, [assetId]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newComment.trim()) return;
    await addComment(assetId, assetType, newComment);
    setNewComment('');
    fetchComments(); // Refetch comments after adding a new one
  };

  const topLevelComments = comments.filter(c => !c.parent_id);
  const repliesByParent = comments.filter(c => c.parent_id).reduce((acc, reply) => {
    acc[reply.parent_id!] = [...(acc[reply.parent_id!] || []), reply];
    return acc;
  }, {} as Record<string, Comment[]>);

  return (
    <div className="comments-panel">
      <h4>Comments</h4>
      {loading && <div>Loading comments...</div>}
      <div className="comment-list">
        {topLevelComments.map(comment => (
          <CommentThread key={comment.id} comment={comment} replies={repliesByParent[comment.id] || []} />
        ))}
      </div>
      <form onSubmit={handleSubmit} className="comment-form">
        <textarea value={newComment} onChange={e => setNewComment(e.target.value)} placeholder="Add a comment..." />
        <button type="submit">Submit</button>
      </form>
    </div>
  );
}