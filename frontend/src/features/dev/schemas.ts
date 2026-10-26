import { z } from 'zod';

// The developer forms check nothing themselves: the API's problems show at their fields.
export const folderSchema = z.object({ name: z.string() });

export type FolderValues = z.infer<typeof folderSchema>;

export const noteSchema = z.object({ title: z.string() });

export type NoteValues = z.infer<typeof noteSchema>;

export const copyNameSchema = z.object({ copyName: z.string() });

export type CopyNameValues = z.infer<typeof copyNameSchema>;
