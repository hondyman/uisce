import { useState, useEffect } from 'react';
import { devError } from '../../utils/devLogger';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Dialog, DialogContent, DialogTrigger } from '@/components/ui/dialog';
import ModalHeader from '../ModalHeader';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Textarea } from '@/components/ui/textarea';
import { Users, Clock, CheckCircle, AlertTriangle, Plus, Filter, Calendar } from 'lucide-react';
import { useNotification } from '../../hooks/useNotification';

interface ReviewStatus {
  metric_id: string;
  metric_name: string;
  review_status: string;
  last_review_date?: string;
  due_date?: string;
  overdue_count: number;
}

interface StewardReview {
  id: string;
  metric_id: string;
  review_period_start: string;
  review_period_end: string;
  reviewer_user_id: string;
  review_type: string;
  overall_rating?: string;
  review_notes?: string;
  action_items: any[];
  status: string;
  due_date?: string;
  completed_at?: string;
  created_at: string;
  updated_at: string;
}

interface StewardReviewPanelProps {
  reviewStatus: ReviewStatus[];
  onRefresh: () => void;
}

export const StewardReviewPanel: React.FC<StewardReviewPanelProps> = ({
  reviewStatus
}) => {
  const [reviews, setReviews] = useState<StewardReview[]>([]);
  const [loading, setLoading] = useState(false);
  const [statusFilter, setStatusFilter] = useState('');
  const [selectedReview, setSelectedReview] = useState<StewardReview | null>(null);
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [showUpdateDialog, setShowUpdateDialog] = useState(false);
  const notification = useNotification();

  useEffect(() => {
    fetchReviews();
  }, []);

  const fetchReviews = async () => {
    setLoading(true);
    try {
      const response = await fetch('/api/pop/reviews');
      if (!response.ok) throw new Error('Failed to fetch reviews');

      const data = await response.json();
      setReviews(data.reviews || []);
    } catch (error) {
      try { devError('Error fetching reviews:', error); } catch {}
    } finally {
      setLoading(false);
    }
  };

  const handleCreateReview = async (reviewData: Partial<StewardReview>) => {
    try {
      const response = await fetch('/api/pop/reviews', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(reviewData),
      });

      if (!response.ok) throw new Error('Failed to create review');

      fetchReviews();
      setShowCreateDialog(false);
    } catch (error) {
      try { devError('Error creating review:', error); } catch {}
      notification.error('Failed to create review');
    }
  };

  const handleUpdateReview = async (reviewId: string, updateData: Partial<StewardReview>) => {
    try {
      const response = await fetch(`/api/pop/reviews/${reviewId}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(updateData),
      });

      if (!response.ok) throw new Error('Failed to update review');

      fetchReviews();
      setShowUpdateDialog(false);
      setSelectedReview(null);
    } catch (error) {
      try { devError('Error updating review:', error); } catch {}
      notification.error('Failed to update review');
    }
  };

  const filteredReviews = reviews.filter(review => {
    if (statusFilter && review.status !== statusFilter) return false;
    return true;
  });

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'completed': return 'text-green-600 bg-green-50 border-green-200';
      case 'in_progress': return 'text-blue-600 bg-blue-50 border-blue-200';
      case 'overdue': return 'text-red-600 bg-red-50 border-red-200';
      case 'pending': return 'text-yellow-600 bg-yellow-50 border-yellow-200';
      default: return 'text-gray-600 bg-gray-50 border-gray-200';
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'completed': return <CheckCircle className="w-4 h-4 text-green-500" />;
      case 'in_progress': return <Clock className="w-4 h-4 text-blue-500" />;
      case 'overdue': return <AlertTriangle className="w-4 h-4 text-red-500" />;
      default: return <Clock className="w-4 h-4 text-gray-500" />;
    }
  };

  const isOverdue = (dueDate?: string) => {
    if (!dueDate) return false;
    return new Date(dueDate) < new Date();
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-gray-900">Steward Review Panel</h2>
          <p className="text-gray-600">Manage data steward reviews and workflows</p>
        </div>
        <div className="flex items-center space-x-2">
          <Button onClick={fetchReviews} variant="outline" disabled={loading}>
            Refresh
          </Button>
          <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
            <DialogTrigger asChild>
              <Button>
                <Plus className="w-4 h-4 mr-2" />
                New Review
              </Button>
            </DialogTrigger>
            <DialogContent className="max-w-2xl">
              <ModalHeader title="Create Steward Review" onClose={() => setShowCreateDialog(false)} />
              <CreateReviewForm
                onSubmit={handleCreateReview}
                onCancel={() => setShowCreateDialog(false)}
              />
            </DialogContent>
          </Dialog>
        </div>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Reviews</CardTitle>
            <Users className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{reviews.length}</div>
            <p className="text-xs text-muted-foreground">
              All steward reviews
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">In Progress</CardTitle>
            <Clock className="h-4 w-4 text-blue-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-blue-600">
              {reviews.filter(r => r.status === 'in_progress').length}
            </div>
            <p className="text-xs text-muted-foreground">
              Currently active
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Completed</CardTitle>
            <CheckCircle className="h-4 w-4 text-green-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-green-600">
              {reviews.filter(r => r.status === 'completed').length}
            </div>
            <p className="text-xs text-muted-foreground">
              This month
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Overdue</CardTitle>
            <AlertTriangle className="h-4 w-4 text-red-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-red-600">
              {reviews.filter(r => isOverdue(r.due_date)).length}
            </div>
            <p className="text-xs text-muted-foreground">
              Need attention
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Review Status Overview */}
      <Card>
        <CardHeader>
          <CardTitle>Review Status by Metric</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {reviewStatus.map((status) => (
              <div key={status.metric_id} className="flex items-center justify-between p-4 border rounded-lg">
                <div>
                  <p className="font-medium">{status.metric_name}</p>
                  <p className="text-sm text-gray-500">
                    Last reviewed: {status.last_review_date ?
                      new Date(status.last_review_date).toLocaleDateString() :
                      'Never'
                    }
                  </p>
                </div>
                <div className="flex items-center space-x-4">
                  <Badge className={getStatusColor(status.review_status)}>
                    {status.review_status.replace('_', ' ')}
                  </Badge>
                  {status.due_date && (
                    <div className="text-right">
                      <div className="text-sm text-gray-500">Due</div>
                      <div className={`text-sm font-medium ${isOverdue(status.due_date) ? 'text-red-600' : ''}`}>
                        {new Date(status.due_date).toLocaleDateString()}
                      </div>
                    </div>
                  )}
                  {status.overdue_count > 0 && (
                    <Badge variant="destructive">
                      {status.overdue_count} overdue
                    </Badge>
                  )}
                </div>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* Filters */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center">
            <Filter className="w-4 h-4 mr-2" />
            Filters
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <Select value={statusFilter} onValueChange={setStatusFilter}>
              <SelectTrigger>
                <SelectValue placeholder="All Statuses" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="">All Statuses</SelectItem>
                <SelectItem value="pending">Pending</SelectItem>
                <SelectItem value="in_progress">In Progress</SelectItem>
                <SelectItem value="completed">Completed</SelectItem>
                <SelectItem value="overdue">Overdue</SelectItem>
              </SelectContent>
            </Select>

            <Button
              variant="outline"
              onClick={() => setStatusFilter('')}
            >
              Clear Filters
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Reviews Table */}
      <Card>
        <CardHeader>
          <CardTitle>
            Steward Reviews ({filteredReviews.length} of {reviews.length})
          </CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Metric</TableHead>
                <TableHead>Review Period</TableHead>
                <TableHead>Reviewer</TableHead>
                <TableHead>Type</TableHead>
                <TableHead>Rating</TableHead>
                <TableHead>Due Date</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredReviews.map((review) => (
                <TableRow key={review.id}>
                  <TableCell>
                    <div className="font-medium">
                      {reviewStatus.find(s => s.metric_id === review.metric_id)?.metric_name || 'Unknown'}
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="text-sm">
                      {new Date(review.review_period_start).toLocaleDateString()} -
                      {new Date(review.review_period_end).toLocaleDateString()}
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="text-sm">{review.reviewer_user_id}</div>
                  </TableCell>
                  <TableCell>
                    <Badge variant="outline">{review.review_type}</Badge>
                  </TableCell>
                  <TableCell>
                    {review.overall_rating ? (
                      <Badge variant={
                        review.overall_rating === 'excellent' ? 'default' :
                        review.overall_rating === 'good' ? 'secondary' :
                        review.overall_rating === 'needs_attention' ? 'destructive' :
                        'outline'
                      }>
                        {review.overall_rating.replace('_', ' ')}
                      </Badge>
                    ) : (
                      <span className="text-gray-400">Not rated</span>
                    )}
                  </TableCell>
                  <TableCell>
                    {review.due_date ? (
                      <div className={`text-sm ${isOverdue(review.due_date) ? 'text-red-600 font-medium' : ''}`}>
                        {new Date(review.due_date).toLocaleDateString()}
                      </div>
                    ) : (
                      <span className="text-gray-400">No due date</span>
                    )}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center">
                      {getStatusIcon(review.status)}
                      <span className="ml-2 capitalize">{review.status.replace('_', ' ')}</span>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center space-x-2">
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => setSelectedReview(review)}
                      >
                        View
                      </Button>
                      {review.status !== 'completed' && (
                        <Button
                          size="sm"
                          onClick={() => {
                            setSelectedReview(review);
                            setShowUpdateDialog(true);
                          }}
                        >
                          Update
                        </Button>
                      )}
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Review Detail Dialog */}
      {selectedReview && !showUpdateDialog && (
        <Dialog open={!!selectedReview} onOpenChange={() => setSelectedReview(null)}>
          <DialogContent className="max-w-2xl">
              <ModalHeader title="Review Details" onClose={() => setShowUpdateDialog(false)} />
            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <h4 className="font-medium mb-2">Review Information</h4>
                  <dl className="space-y-1 text-sm">
                    <div><dt className="inline font-medium">Metric:</dt> <dd className="inline ml-2">
                      {reviewStatus.find(s => s.metric_id === selectedReview.metric_id)?.metric_name || 'Unknown'}
                    </dd></div>
                    <div><dt className="inline font-medium">Reviewer:</dt> <dd className="inline ml-2">{selectedReview.reviewer_user_id}</dd></div>
                    <div><dt className="inline font-medium">Type:</dt> <dd className="inline ml-2">{selectedReview.review_type}</dd></div>
                    <div><dt className="inline font-medium">Status:</dt> <dd className="inline ml-2">
                      <Badge className={getStatusColor(selectedReview.status)}>
                        {selectedReview.status.replace('_', ' ')}
                      </Badge>
                    </dd></div>
                    <div><dt className="inline font-medium">Created:</dt> <dd className="inline ml-2">{new Date(selectedReview.created_at).toLocaleString()}</dd></div>
                  </dl>
                </div>
                <div>
                  <h4 className="font-medium mb-2">Timeline</h4>
                  <dl className="space-y-1 text-sm">
                    <div><dt className="inline font-medium">Period:</dt> <dd className="inline ml-2">
                      {new Date(selectedReview.review_period_start).toLocaleDateString()} -
                      {new Date(selectedReview.review_period_end).toLocaleDateString()}
                    </dd></div>
                    <div><dt className="inline font-medium">Due Date:</dt> <dd className="inline ml-2">
                      {selectedReview.due_date ? new Date(selectedReview.due_date).toLocaleDateString() : 'Not set'}
                    </dd></div>
                    <div><dt className="inline font-medium">Completed:</dt> <dd className="inline ml-2">
                      {selectedReview.completed_at ? new Date(selectedReview.completed_at).toLocaleString() : 'Not completed'}
                    </dd></div>
                    <div><dt className="inline font-medium">Rating:</dt> <dd className="inline ml-2">
                      {selectedReview.overall_rating || 'Not rated'}
                    </dd></div>
                  </dl>
                </div>
              </div>

              {selectedReview.review_notes && (
                <div>
                  <h4 className="font-medium mb-2">Review Notes</h4>
                  <p className="text-sm text-gray-600 bg-gray-50 p-3 rounded">
                    {selectedReview.review_notes}
                  </p>
                </div>
              )}

              {selectedReview.action_items && selectedReview.action_items.length > 0 && (
                <div>
                  <h4 className="font-medium mb-2">Action Items</h4>
                  <div className="space-y-2">
                    {selectedReview.action_items.map((item: any, index: number) => (
                      <div key={index} className="flex items-center space-x-2 p-2 bg-gray-50 rounded">
                        <input
                          type="checkbox"
                          checked={item.completed || false}
                          readOnly
                          className="rounded"
                          title={`Action item: ${item.description}`}
                        />
                        <span className="text-sm">{item.description}</span>
                        {item.due_date && (
                          <Badge variant="outline" className="text-xs">
                            <Calendar className="w-3 h-3 mr-1" />
                            {new Date(item.due_date).toLocaleDateString()}
                          </Badge>
                        )}
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </DialogContent>
        </Dialog>
      )}

      {/* Update Review Dialog */}
      {selectedReview && showUpdateDialog && (
        <UpdateReviewDialog
          review={selectedReview}
          onUpdate={handleUpdateReview}
          onCancel={() => {
            setShowUpdateDialog(false);
            setSelectedReview(null);
          }}
        />
      )}
    </div>
  );
};

// Create Review Form Component
interface CreateReviewFormProps {
  onSubmit: (reviewData: Partial<StewardReview>) => void;
  onCancel: () => void;
}

const CreateReviewForm: React.FC<CreateReviewFormProps> = ({ onSubmit, onCancel }) => {
  const [formData, setFormData] = useState({
    metric_id: '',
    review_period_start: '',
    review_period_end: '',
    reviewer_user_id: '',
    review_type: 'monthly',
    overall_rating: '',
    review_notes: '',
    due_date: '',
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSubmit(formData);
  };

  const handleChange = (field: string, value: string) => {
    setFormData(prev => ({ ...prev, [field]: value }));
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="grid grid-cols-2 gap-4">
        <div>
          <label className="block text-sm font-medium mb-1">Metric ID</label>
          <input
            type="text"
            value={formData.metric_id}
            onChange={(e) => handleChange('metric_id', e.target.value)}
            className="w-full p-2 border border-gray-300 rounded"
            placeholder="Enter metric ID"
            title="Metric ID"
            required
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">Reviewer</label>
          <input
            type="text"
            value={formData.reviewer_user_id}
            onChange={(e) => handleChange('reviewer_user_id', e.target.value)}
            className="w-full p-2 border border-gray-300 rounded"
            placeholder="Enter reviewer user ID"
            title="Reviewer User ID"
            required
          />
        </div>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div>
          <label className="block text-sm font-medium mb-1">Review Period Start</label>
          <input
            type="date"
            value={formData.review_period_start}
            onChange={(e) => handleChange('review_period_start', e.target.value)}
            className="w-full p-2 border border-gray-300 rounded"
            title="Review Period Start Date"
            required
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">Review Period End</label>
          <input
            type="date"
            value={formData.review_period_end}
            onChange={(e) => handleChange('review_period_end', e.target.value)}
            className="w-full p-2 border border-gray-300 rounded"
            title="Review Period End Date"
            required
          />
        </div>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div>
          <label className="block text-sm font-medium mb-1">Review Type</label>
          <select
            value={formData.review_type}
            onChange={(e) => handleChange('review_type', e.target.value)}
            className="w-full p-2 border border-gray-300 rounded"
            title="Review Type"
          >
            <option value="monthly">Monthly</option>
            <option value="quarterly">Quarterly</option>
            <option value="annual">Annual</option>
            <option value="ad_hoc">Ad Hoc</option>
          </select>
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">Due Date</label>
          <input
            type="date"
            value={formData.due_date}
            onChange={(e) => handleChange('due_date', e.target.value)}
            className="w-full p-2 border border-gray-300 rounded"
            title="Due Date"
          />
        </div>
      </div>

      <div>
        <label className="block text-sm font-medium mb-1">Overall Rating</label>
        <select
          value={formData.overall_rating}
          onChange={(e) => handleChange('overall_rating', e.target.value)}
          className="w-full p-2 border border-gray-300 rounded"
          title="Overall Rating"
        >
          <option value="">Select rating</option>
          <option value="excellent">Excellent</option>
          <option value="good">Good</option>
          <option value="satisfactory">Satisfactory</option>
          <option value="needs_attention">Needs Attention</option>
          <option value="critical">Critical</option>
        </select>
      </div>

      <div>
        <label className="block text-sm font-medium mb-1">Review Notes</label>
        <Textarea
          value={formData.review_notes}
          onChange={(e) => handleChange('review_notes', e.target.value)}
          rows={4}
          placeholder="Enter review notes..."
        />
      </div>

      <div className="flex justify-end space-x-2">
        <Button type="button" variant="outline" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit">
          Create Review
        </Button>
      </div>
    </form>
  );
};

// Update Review Dialog Component
interface UpdateReviewDialogProps {
  review: StewardReview;
  onUpdate: (reviewId: string, updateData: Partial<StewardReview>) => void;
  onCancel: () => void;
}

const UpdateReviewDialog: React.FC<UpdateReviewDialogProps> = ({
  review,
  onUpdate,
  onCancel
}) => {
  const [formData, setFormData] = useState({
    overall_rating: review.overall_rating || '',
    review_notes: review.review_notes || '',
    status: review.status,
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const updateData: Partial<StewardReview> = {
      ...formData,
      updated_at: new Date().toISOString(),
    };

    if (formData.status === 'completed') {
      updateData.completed_at = new Date().toISOString();
    }

    onUpdate(review.id, updateData);
  };

  const handleChange = (field: string, value: string) => {
    setFormData(prev => ({ ...prev, [field]: value }));
  };

  return (
    <Dialog open={true} onOpenChange={onCancel}>
  <DialogContent>
  <ModalHeader title="Update Steward Review" onClose={onCancel} />
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">Overall Rating</label>
            <select
              value={formData.overall_rating}
              onChange={(e) => handleChange('overall_rating', e.target.value)}
              className="w-full p-2 border border-gray-300 rounded"
              title="Overall Rating"
            >
              <option value="">Select rating</option>
              <option value="excellent">Excellent</option>
              <option value="good">Good</option>
              <option value="satisfactory">Satisfactory</option>
              <option value="needs_attention">Needs Attention</option>
              <option value="critical">Critical</option>
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">Status</label>
            <select
              value={formData.status}
              onChange={(e) => handleChange('status', e.target.value)}
              className="w-full p-2 border border-gray-300 rounded"
              title="Review Status"
            >
              <option value="pending">Pending</option>
              <option value="in_progress">In Progress</option>
              <option value="completed">Completed</option>
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">Review Notes</label>
            <Textarea
              value={formData.review_notes}
              onChange={(e) => handleChange('review_notes', e.target.value)}
              rows={4}
              placeholder="Update review notes..."
            />
          </div>

          <div className="flex justify-end space-x-2">
            <Button type="button" variant="outline" onClick={onCancel}>
              Cancel
            </Button>
            <Button type="submit">
              Update Review
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
};
