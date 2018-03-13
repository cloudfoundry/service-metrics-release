require 'tempfile'

describe 'service metrics' do

  before(:all) do
    @outFile = Tempfile.new('smetrics')
    @pid = spawn(
      {
        "DOPPLER_ADDR" => doppler_address,
        "CF_ACCESS_TOKEN" => cf_auth_token
      },
      'firehose',
      [:out, :err] => [@outFile.path, 'w']
    )

    @metric_entry = find_metric_entry(@outFile)
    expect(@metric_entry).to_not be_nil
  end

  after(:all) do
    Process.kill("INT", @pid)
    @outFile.unlink
  end

  it "it emits metrics" do
    expect(@metric_entry).to match(/name:"service-dummy"/)
  end

  it "it emits the origin" do
    expect(@metric_entry).to match(/origin:"#{origin}"/)
  end

  it "it emits ip" do
    expect(@metric_entry).to match(/ip:"/)
  end

  it "it emits job" do
    expect(@metric_entry).to match(/job:"service-metrics"/)
  end

  it "it emits index" do
    expect(@metric_entry).to match(/index:".*"/)
  end

  it "it emits deployment" do
    expect(@metric_entry).to match(/deployment:"#{deployment_name}"/)
  end

  def find_metric_entry(firehose_out_file)
    metric_entry = nil

    60.times do
      firehose_out_file.read.lines.each do | line |
        line.force_encoding("utf-8")
        if line =~ /origin:"#{origin}".*name:"service-dummy"/
          metric_entry = line
          break
        end
      end
      metric_entry ? break : sleep(1)
    end

    return metric_entry
  end
end
