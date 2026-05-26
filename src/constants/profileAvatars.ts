export const PROFILE_AVATAR_FILENAMES = [
  'Alien.png',
  'Clown Face.png',
  'Cold Face.png',
  'Face With Symbols On Mouth.png',
  'Face With Thermometer.png',
  'Nerd Face.png',
  'Pleading Face.png',
  'Rolling On The Floor Laughing.png',
  'Skull.png',
  'Slightly Smiling Face.png',
  'Smiling Face With Hearts.png',
  'Smiling Face With Horns.png',
  'Star Struck.png',
  'Winking Face With Tongue.png',
  'Winking Face.png',
] as const;

export const DEFAULT_PROFILE_AVATAR_FILENAME = PROFILE_AVATAR_FILENAMES[0];

export function avatarUrlFromFilename(filename: string): string {
  return `/img/FullPic/avatars/${encodeURIComponent(filename)}`;
}
