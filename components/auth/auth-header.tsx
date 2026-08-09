type Props = {
  title: string;
  description: string;
};

export const AuthHeader = ({ title, description }: Props) => {
  return (
    <div className="space-y-2 text-center">
      <h1 className="text-3xl font-bold text-[#2E2A47]">{title}</h1>
      <p className="text-base text-[#7E8CA0]">{description}</p>
    </div>
  );
};
