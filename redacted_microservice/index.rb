require "grape"
require "confidential_info_redactor"
require "cld"

def redact(text)
  if text then
  lang = CLD.detect_language(text)
  tokens = ConfidentialInfoRedactor::Extractor.new(language: lang[:code]).extract(text)
  redacted = ConfidentialInfoRedactor::Redactor.new(tokens: tokens , number_text: '⬛⬛⬛', date_text: '⬛⬛/⬛⬛/⬛⬛⬛⬛', token_text: '⬛⬛⬛⬛⬛⬛⬛⬛').redact(text)
  else
    redacted=""
  end
  redacted

end

module Text
  class API < Grape::API
    version 'v1', using: :header, vendor: 'platform_sh'
    content_type :txt, 'text/plain'
    content_type :json, 'application/json'
    
    format :json
    params do
       optional :text, type: String
     end

    desc "Returns redacted text"
    get "" do
      content_type 'text/plain'
      redact(params[:text])
    end
    
    desc 'Returns redacted text'
    post "" do
      content_type 'text/plain'
      redact(params[:text])
    end

    desc 'Discovery'
    get "discover" do
      { "name"=>"redacted", "type"=>"*ast.Text", "flags"=>{ "composable"=>true } }
    end
  end
  
end
