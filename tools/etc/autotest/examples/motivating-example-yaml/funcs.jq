def port_of($service):
  refexpr(".ports.\($service)");

def url_of($service):
  refexpr(".urls.\($service)");
