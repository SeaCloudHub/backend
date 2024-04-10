import { Namespace, SubjectSet, Context } from '@ory/keto-namespace-types';

class User implements Namespace {
	related: {
		manager: User[];
	};
}

class Group implements Namespace {
	related: {
		members: (User | Group)[];
	};
}

class Directory implements Namespace {
	related: {
		parents: Directory[];
		viewers: (User | SubjectSet<Group, 'members'>)[];
		editors: (User | SubjectSet<Group, 'members'>)[];
		owners: (User | SubjectSet<Group, 'members'>)[];
		managers: (User | SubjectSet<Group, 'members'>)[]; // admin
	};

	permits = {
		// View is allowed if the user is a viewer, editor, owner, manager, or has permission to view the parent
		view: (ctx: Context): boolean =>
			this.related.viewers.includes(ctx.subject) ||
			this.related.editors.includes(ctx.subject) ||
			this.related.owners.includes(ctx.subject) ||
			this.related.managers.includes(ctx.subject) ||
			this.related.parents.traverse((p) => p.permits.view(ctx)),

		// Edit is allowed if the user is an owner, editor, manager, or has permission to edit the parent
		// Those who can edit can also share the directory
		edit: (ctx: Context): boolean =>
			this.related.owners.includes(ctx.subject) ||
			this.related.editors.includes(ctx.subject) ||
			this.related.managers.includes(ctx.subject) ||
			this.related.parents.traverse((p) => p.permits.edit(ctx)),

		// Delete is allowed if the user has permission to move to trash, or is a parent editor
		delete: (ctx: Context): boolean =>
			this.permits.move_to_trash(ctx) ||
			this.related.parents.traverse((p) =>
				p.related.editors.includes(ctx.subject)
			),

		// Move to trash is allowed if the user is an owner, manager, or has permission to move to trash the parent
		move_to_trash: (ctx: Context): boolean =>
			this.related.owners.includes(ctx.subject) ||
			this.related.managers.includes(ctx.subject) ||
			this.related.parents.traverse((p) => p.permits.move_to_trash(ctx)),
	};
}

class File implements Namespace {
	related: {
		parents: Directory[];
		viewers: (User | SubjectSet<Group, 'members'>)[];
		editors: (User | SubjectSet<Group, 'members'>)[];
		owners: (User | SubjectSet<Group, 'members'>)[];
		managers: (User | SubjectSet<Group, 'members'>)[]; // admin
	};

	permits = {
		// View is allowed if the user is a viewer, editor, owner, manager, or has permission to view the parent
		view: (ctx: Context): boolean =>
			this.related.viewers.includes(ctx.subject) ||
			this.related.editors.includes(ctx.subject) ||
			this.related.owners.includes(ctx.subject) ||
			this.related.managers.includes(ctx.subject) ||
			this.related.parents.traverse((p) => p.permits.view(ctx)),

		// Edit is allowed if the user is an owner, editor, manager, or has permission to edit the parent
		// Those who can edit can also share the directory
		edit: (ctx: Context): boolean =>
			this.related.owners.includes(ctx.subject) ||
			this.related.editors.includes(ctx.subject) ||
			this.related.managers.includes(ctx.subject) ||
			this.related.parents.traverse((p) => p.permits.edit(ctx)),

		// Delete is allowed if the user has permission to move to trash, or is a parent editor
		delete: (ctx: Context): boolean =>
			this.permits.move_to_trash(ctx) ||
			this.related.parents.traverse((p) =>
				p.related.editors.includes(ctx.subject)
			),

		// Move to trash is allowed if the user is an owner, manager, or has permission to move to trash the parent
		move_to_trash: (ctx: Context): boolean =>
			this.related.owners.includes(ctx.subject) ||
			this.related.managers.includes(ctx.subject) ||
			this.related.parents.traverse((p) => p.permits.move_to_trash(ctx)),
	};
}
