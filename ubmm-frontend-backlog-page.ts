// frontend/src/app/backlog/page.tsx

import React from 'react';
import { Metadata } from 'next';
import BacklogBoard from '@/components/backlog/BacklogBoard';
import BacklogHeader from '@/components/backlog/BacklogHeader';
import BacklogFilter from '@/components/backlog/BacklogFilter';
import BacklogMetrics from '@/components/backlog/BacklogMetrics';

export const metadata: Metadata = {
  title: 'Backlog | UBMM',
  description: 'Manage your product backlog with the Ultimate Backlog Management & Monitor',
};

export default async function BacklogPage() {
  return (
    <div className="flex flex-col h-full">
      <BacklogHeader />
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-6">
        <BacklogMetrics />
      </div>
      <div className="flex flex-col space-y-4 flex-grow">
        <BacklogFilter />
        <div className="flex-grow overflow-hidden bg-white rounded-lg shadow">
          <BacklogBoard />
        </div>
      </div>
    </div>
  );
}

// frontend/src/components/backlog/BacklogHeader.tsx

'use client';

import React from 'react';
import { Plus, Filter, Download, Upload, RefreshCw } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { CreateItemDialog } from '@/components/backlog/CreateItemDialog';

export default function BacklogHeader() {
  const [isCreateDialogOpen, setIsCreateDialogOpen] = React.useState(false);
  
  return (
    <div className="flex flex-col md:flex-row justify-between items-start md:items-center mb-6 gap-4">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Product Backlog</h1>
        <p className="text-sm text-gray-500">
          Manage your product backlog using the iceberg model
        </p>
      </div>
      <div className="flex flex-wrap gap-2">
        <Button
          variant="outline"
          size="sm"
          className="flex items-center gap-1"
          onClick={() => {}}
        >
          <RefreshCw size={16} />
          <span className="hidden sm:inline">Refresh</span>
        </Button>
        <Button
          variant="outline"
          size="sm"
          className="flex items-center gap-1"
          onClick={() => {}}
        >
          <Filter size={16} />
          <span className="hidden sm:inline">Filter</span>
        </Button>
        <Button
          variant="outline"
          size="sm"
          className="flex items-center gap-1"
          onClick={() => {}}
        >
          <Download size={16} />
          <span className="hidden sm:inline">Export</span>
        </Button>
        <Button
          variant="outline"
          size="sm"
          className="flex items-center gap-1"
          onClick={() => {}}
        >
          <Upload size={16} />
          <span className="hidden sm:inline">Import</span>
        </Button>
        <Button
          variant="default"
          size="sm"
          className="flex items-center gap-1"
          onClick={() => setIsCreateDialogOpen(true)}
        >
          <Plus size={16} />
          <span>New Item</span>
        </Button>
      </div>
      
      <CreateItemDialog 
        isOpen={isCreateDialogOpen} 
        onClose={() => setIsCreateDialogOpen(false)} 
      />
    </div>
  );
}

// frontend/src/components/backlog/BacklogMetrics.tsx

'use client';

import React from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Progress } from '@/components/ui/progress';
import { AlertTriangle, CheckCircle2, Clock, GitPullRequest } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { fetchBacklogMetrics } from '@/lib/api/backlog';

export default function BacklogMetrics() {
  const { data: metrics, isLoading, error } = useQuery({
    queryKey: ['backlogMetrics'],
    queryFn: fetchBacklogMetrics,
    refetchInterval: 60000, // Refetch every minute
  });

  if (isLoading) {
    return (
      <>
        <MetricSkeleton />
        <MetricSkeleton />
        <MetricSkeleton />
        <MetricSkeleton />
      </>
    );
  }

  if (error || !metrics) {
    return (
      <Card className="col-span-4">
        <CardContent className="pt-6">
          <div className="text-center text-red-500">
            <AlertTriangle className="mx-auto h-8 w-8 mb-2" />
            <p>Failed to load metrics</p>
          </div>
        </CardContent>
      </Card>
    );
  }

  const healthStatusColors = {
    'HEALTHY': 'text-green-500',
    'AVERAGE': 'text-yellow-500',
    'WARNING': 'text-orange-500',
    'AT_RISK': 'text-red-500'
  };

  const healthStatusColor = healthStatusColors[metrics.healthStatus as keyof typeof healthStatusColors] || 'text-gray-500';

  return (
    <>
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium">Backlog Items</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">{metrics.totalItems}</div>
          <div className="text-xs text-muted-foreground mt-1">
            {metrics.epicCount} Epics • {metrics.featureCount} Features • {metrics.storyCount} Stories
          </div>
          <div className="mt-3">
            <div className="flex items-center justify-between text-xs mb-1">
              <span>Iceberg Ratio</span>
              <span>{Math.round(metrics.icebergRatio * 100)}%</span>
            </div>
            <Progress value={metrics.icebergRatio * 100} className="h-1" />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium">Health Status</CardTitle>
        </CardHeader>
        <CardContent>
          <div className={`text-2xl font-bold ${healthStatusColor}`}>
            {metrics.healthStatus.replace('_', ' ')}
          </div>
          <div className="flex items-center text-xs text-muted-foreground mt-1">
            <Clock className="h-3 w-3 mr-1" />
            <span>{Math.round(metrics.averageAge)} days avg. age</span>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium">Work in Progress</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">{metrics.wipCount}</div>
          <div className="flex items-center text-xs text-muted-foreground mt-1">
            <GitPullRequest className="h-3 w-3 mr-1" />
            <span>Active items being worked on</span>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium">Delivery Metrics</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">{Math.round(metrics.leadTimeDays)} days</div>
          <div className="flex items-center text-xs text-muted-foreground mt-1">
            <CheckCircle2 className="h-3 w-3 mr-1" />
            <span>{metrics.throughputLast30Days} items completed in 30 days</span>
          </div>
        </CardContent>
      </Card>
    </>
  );
}

function MetricSkeleton() {
  return (
    <Card>
      <CardHeader className="pb-2">
        <div className="h-4 w-24 bg-gray-200 rounded animate-pulse"></div>
      </CardHeader>
      <CardContent>
        <div className="h-8 w-16 bg-gray-200 rounded animate-pulse mb-2"></div>
        <div className="h-3 w-full bg-gray-200 rounded animate-pulse"></div>
      </CardContent>
    </Card>
  );
}

// frontend/src/components/backlog/BacklogFilter.tsx

'use client';

import React from 'react';
import { Search, X } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Badge } from '@/components/ui/badge';
import { useBacklogFilters } from '@/hooks/useBacklogFilters';

export default function BacklogFilter() {
  const { 
    filters, 
    setTypeFilter, 
    setStatusFilter, 
    setAssigneeFilter,
    setSearchQuery,
    clearFilters,
    hasActiveFilters
  } = useBacklogFilters();

  return (
    <div className="bg-white p-4 rounded-lg shadow">
      <div className="flex flex-col md:flex-row gap-3">
        <div className="relative flex-grow">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-500 h-4 w-4" />
          <Input
            placeholder="Search backlog items..."
            className="pl-9"
            value={filters.searchQuery || ''}
            onChange={(e) => setSearchQuery(e.target.value)}
          />
        </div>
        
        <div className="flex flex-wrap gap-2">
          <Select value={filters.type || ''} onValueChange={setTypeFilter}>
            <SelectTrigger className="w-[120px]">
              <SelectValue placeholder="Type" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="">All Types</SelectItem>
              <SelectItem value="EPIC">Epic</SelectItem>
              <SelectItem value="FEATURE">Feature</SelectItem>
              <SelectItem value="STORY">Story</SelectItem>
            </SelectContent>
          </Select>
          
          <Select value={filters.status || ''} onValueChange={setStatusFilter}>
            <SelectTrigger className="w-[140px]">
              <SelectValue placeholder="Status" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="">All Statuses</SelectItem>
              <SelectItem value="NEW">New</SelectItem>
              <SelectItem value="READY">Ready</SelectItem>
              <SelectItem value="IN_PROGRESS">In Progress</SelectItem>
              <SelectItem value="BLOCKED">Blocked</SelectItem>
              <SelectItem value="DONE">Done</SelectItem>
            </SelectContent>
          </Select>
          
          <Select value={filters.assignee || ''} onValueChange={setAssigneeFilter}>
            <SelectTrigger className="w-[140px]">
              <SelectValue placeholder="Assignee" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="">All Assignees</SelectItem>
              <SelectItem value="me">Assigned to Me</SelectItem>
              <SelectItem value="unassigned">Unassigned</SelectItem>
              {/* We would typically fetch users from the API */}
            </SelectContent>
          </Select>
        </div>
      </div>
      
      {hasActiveFilters && (
        <div className="flex flex-wrap gap-2 mt-3 items-center">
          <span className="text-xs text-gray-500">Active filters:</span>
          
          {filters.type && (
            <Badge variant="outline" className="text-xs">
              Type: {filters.type}
              <X 
                className="h-3 w-3 ml-1 cursor-pointer" 
                onClick={() => setTypeFilter('')}
              />
            </Badge>
          )}
          
          {filters.status && (
            <Badge variant="outline" className="text-xs">
              Status: {filters.status}
              <X 
                className="h-3 w-3 ml-1 cursor-pointer" 
                onClick={() => setStatusFilter('')}
              />
            </Badge>
          )}
          
          {filters.assignee && (
            <Badge variant="outline" className="text-xs">
              Assignee: {filters.assignee === 'me' ? 'Me' : filters.assignee === 'unassigned' ? 'Unassigned' : filters.assignee}
              <X 
                className="h-3 w-3 ml-1 cursor-pointer" 
                onClick={() => setAssigneeFilter('')}
              />
            </Badge>
          )}
          
          {filters.searchQuery && (
            <Badge variant="outline" className="text-xs">
              Search: {filters.searchQuery}
              <X 
                className="h-3 w-3 ml-1 cursor-pointer" 
                onClick={() => setSearchQuery('')}
              />
            </Badge>
          )}
          
          <Button
            variant="ghost"
            size="sm"
            className="text-xs h-6 px-2"
            onClick={clearFilters}
          >
            Clear all
          </Button>
        </div>
      )}
    </div>
  );
}

// frontend/src/components/backlog/BacklogBoard.tsx

'use client';

import React, { useState } from 'react';
import { AlertTriangle, MoreHorizontal, Edit2, Trash2, ExternalLink, Copy } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { useBacklogFilters } from '@/hooks/useBacklogFilters';
import { useQuery } from '@tanstack/react-query';
import { fetchBacklogItems } from '@/lib/api/backlog';
import { EditItemDialog } from '@/components/backlog/EditItemDialog';
import { BacklogItem } from '@/types/backlog';
import { DeleteConfirmDialog } from '@/components/shared/DeleteConfirmDialog';

export default function BacklogBoard() {
  const { filters } = useBacklogFilters();
  const [editingItem, setEditingItem] = useState<BacklogItem | null>(null);
  const [deletingItem, setDeletingItem] = useState<BacklogItem | null>(null);
  
  const { data, isLoading, error } = useQuery({
    queryKey: ['backlogItems', filters],
    queryFn: () => fetchBacklogItems(filters),
    keepPreviousData: true,
  });

  if (isLoading) {
    return (
      <div className="p-4">
        <div className="space-y-3">
          <ItemSkeleton />
          <ItemSkeleton />
          <ItemSkeleton />
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-8 text-center">
        <AlertTriangle className="h-8 w-8 text-red-500 mx-auto mb-2" />
        <h3 className="text-lg font-medium">Failed to load backlog items</h3>
        <p className="text-sm text-gray-500 mt-1">Please try again later</p>
        <Button variant="outline" className="mt-4" onClick={() => window.location.reload()}>
          Retry
        </Button>
      </div>
    );
  }

  if (!data?.items.length) {
    return (
      <div className="p-8 text-center">
        <h3 className="text-lg font-medium">No items found</h3>
        <p className="text-sm text-gray-500 mt-1">
          {Object.values(filters).some(Boolean)
            ? "Try adjusting your filters"
            : "Create your first backlog item to get started"}
        </p>
      </div>
    );
  }

  return (
    <div className="relative overflow-auto">
      <table className="w-full text-sm">
        <thead className="bg-gray-50 text-xs uppercase text-gray-500 sticky top-0">
          <tr>
            <th className="px-4 py-3 text-left w-10">#</th>
            <th className="px-4 py-3 text-left">Title</th>
            <th className="px-4 py-3 text-left w-24">Type</th>
            <th className="px-4 py-3 text-left w-32">Status</th>
            <th className="px-4 py-3 text-left w-20">Points</th>
            <th className="px-4 py-3 text-left w-36">Assignee</th>
            <th className="px-4 py-3 text-left w-10"></th>
          </tr>
        </thead>
        <tbody className="divide-y divide-gray-200">
          {data.items.map((item, index) => (
            <tr key={item.id} className="hover:bg-gray-50">
              <td className="px-4 py-3 text-gray-500">{index + 1}</td>
              <td className="px-4 py-3 font-medium">
                <div className="flex flex-col">
                  <span className="text-gray-900">{item.title}</span>
                  {item.tags.length > 0 && (
                    <div className="flex flex-wrap gap-1 mt-1">
                      {item.tags.map((tag) => (
                        <Badge key={tag} variant="secondary" className="text-xs px-1 py-0">
                          {tag}
                        </Badge>
                      ))}
                    </div>
                  )}
                </div>
              </td>
              <td className="px-4 py-3">
                <TypeBadge type={item.type} />
              </td>
              <td className="px-4 py-3">
                <StatusBadge status={item.status} />
              </td>
              <td className="px-4 py-3 text-center">
                {item.storyPoints > 0 ? item.storyPoints : '-'}
              </td>
              <td className="px-4 py-3">
                {item.assignee ? (
                  <div className="flex items-center">
                    <div className="h-6 w-6 rounded-full bg-gray-300 flex items-center justify-center text-xs mr-2">
                      {item.assignee.charAt(0).toUpperCase()}
                    </div>
                    <span className="truncate max-w-[100px]">{item.assignee}</span>
                  </div>
                ) : (
                  <span className="text-gray-400">Unassigned</span>
                )}
              </td>
              <td className="px-4 py-3">
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
                      <MoreHorizontal className="h-4 w-4" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuItem onClick={() => setEditingItem(item)}>
                      <Edit2 className="mr-2 h-4 w-4" />
                      <span>Edit</span>
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => {}}>
                      <Copy className="mr-2 h-4 w-4" />
                      <span>Duplicate</span>
                    </DropdownMenuItem>
                    {Object.keys(item.externalIds).length > 0 && (
                      <>
                        <DropdownMenuSeparator />
                        {Object.entries(item.externalIds).map(([system, id]) => (
                          <DropdownMenuItem key={system}>
                            <ExternalLink className="mr-2 h-4 w-4" />
                            <span>
                              Open in {system}: {id}
                            </span>
                          </DropdownMenuItem>
                        ))}
                      </>
                    )}
                    <DropdownMenuSeparator />
                    <DropdownMenuItem
                      className="text-red-600"
                      onClick={() => setDeletingItem(item)}
                    >
                      <Trash2 className="mr-2 h-4 w-4" />
                      <span>Delete</span>
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      {editingItem && (
        <EditItemDialog
          item={editingItem}
          isOpen={!!editingItem}
          onClose={() => setEditingItem(null)}
        />
      )}

      {deletingItem && (
        <DeleteConfirmDialog
          title="Delete Backlog Item"
          description={`Are you sure you want to delete "${deletingItem.title}"? This action cannot be undone.`}
          isOpen={!!deletingItem}
          onClose={() => setDeletingItem(null)}
          onConfirm={() => {
            // Handle deletion logic
            setDeletingItem(null);
          }}
        />
      )}
    </div>
  );
}

function TypeBadge({ type }: { type: string }) {
  const typeProps = {
    'EPIC': { color: 'bg-purple-100 text-purple-800', label: 'Epic' },
    'FEATURE': { color: 'bg-blue-100 text-blue-800', label: 'Feature' },
    'STORY': { color: 'bg-green-100 text-green-800', label: 'Story' },
  };

  const { color, label } = typeProps[type as keyof typeof typeProps] || 
    { color: 'bg-gray-100 text-gray-800', label: type };

  return (
    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${color}`}>
      {label}
    </span>
  );
}

function StatusBadge({ status }: { status: string }) {
  const statusProps = {
    'NEW': { color: 'bg-gray-100 text-gray-800', label: 'New' },
    'READY': { color: 'bg-blue-100 text-blue-800', label: 'Ready' },
    'IN_PROGRESS': { color: 'bg-yellow-100 text-yellow-800', label: 'In Progress' },
    'BLOCKED': { color: 'bg-red-100 text-red-800', label: 'Blocked' },
    'DONE': { color: 'bg-green-100 text-green-800', label: 'Done' },
  };

  const { color, label } = statusProps[status as keyof typeof statusProps] || 
    { color: 'bg-gray-100 text-gray-800', label: status };

  return (
    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${color}`}>
      {label}
    </span>
  );
}

function ItemSkeleton() {
  return (
    <div className="border border-gray-200 rounded p-4 animate-pulse">
      <div className="h-5 bg-gray-200 rounded w-1/3 mb-3"></div>
      <div className="h-4 bg-gray-200 rounded w-1/2 mb-2"></div>
      <div className="flex gap-2">
        <div className="h-4 bg-gray-200 rounded w-16"></div>
        <div className="h-4 bg-gray-200 rounded w-20"></div>
      </div>
    </div>
  );
}
