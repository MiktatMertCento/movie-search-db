import { z } from 'zod';

export const searchSchema = z.object({
    query: z.string().min(2, 'En az 2 karakter giriniz').max(100, 'Çok uzun bir arama yaptınız'),
});

export type SearchFormData = z.infer<typeof searchSchema>;

export interface Movie {
    ID: number;
    TmdbID: string;
    Title: string;
    Tag: string;
    Ov: string;
    Post: string;
    Vote: number;
    Sim: number;
}